package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/zajca/zfaktury/internal/config"
)

// Attachment represents a file attached to an email message.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// EmailMessage holds all data for sending an email.
type EmailMessage struct {
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	BodyHTML    string
	BodyText    string
	Attachments []Attachment
}

// EmailSender sends emails via SMTP.
type EmailSender struct {
	cfg config.SMTPConfig
}

// NewEmailSender creates a new EmailSender with the provided SMTP config.
func NewEmailSender(cfg config.SMTPConfig) *EmailSender {
	return &EmailSender{cfg: cfg}
}

// IsConfigured returns true when the SMTP host is set.
func (s *EmailSender) IsConfigured() bool {
	return s.cfg.Host != ""
}

// Send delivers the message via SMTP. It checks ctx for cancellation before
// dialing but the underlying net/smtp library does not support context natively.
func (s *EmailSender) Send(ctx context.Context, msg EmailMessage) error {
	if !s.IsConfigured() {
		return fmt.Errorf("email sender: SMTP not configured (host is empty)")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	raw, err := buildRawMessage(s.cfg.From, msg)
	if err != nil {
		return fmt.Errorf("email sender: building message: %w", err)
	}

	recipients := collectRecipients(msg)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	slog.Info("sending email",
		"to", msg.To,
		"subject", msg.Subject,
		"smtp_host", s.cfg.Host,
		"smtp_port", s.cfg.Port,
	)

	if err := s.sendViaSMTP(addr, recipients, raw); err != nil {
		return fmt.Errorf("email sender: %w", err)
	}

	slog.Info("email sent successfully", "to", msg.To, "subject", msg.Subject)
	return nil
}

// sendViaSMTP handles three connection modes:
//   - port 465: implicit TLS
//   - port 587: STARTTLS
//   - anything else: plain SMTP
func (s *EmailSender) sendViaSMTP(addr string, recipients []string, raw []byte) error {
	from := s.cfg.From
	auth := s.buildAuth()

	switch s.cfg.Port {
	case 465:
		return s.sendTLS(addr, from, recipients, raw, auth)
	case 587:
		return s.sendSTARTTLS(addr, from, recipients, raw, auth)
	default:
		return s.sendPlain(addr, from, recipients, raw, auth)
	}
}

func (s *EmailSender) buildAuth() smtp.Auth {
	if s.cfg.Username == "" {
		return nil
	}
	return smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
}

// sendTLS connects with implicit TLS (port 465).
func (s *EmailSender) sendTLS(addr, from string, recipients []string, raw []byte, auth smtp.Auth) error {
	tlsCfg := &tls.Config{
		ServerName: s.cfg.Host,
		MinVersion: tls.VersionTLS12,
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("TLS dial %s: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP client over TLS: %w", err)
	}
	defer client.Close()
	return sendWithClient(client, from, recipients, raw, auth)
}

// sendSTARTTLS connects plain then upgrades to TLS.
func (s *EmailSender) sendSTARTTLS(addr, from string, recipients []string, raw []byte, auth smtp.Auth) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("SMTP dial %s: %w", addr, err)
	}
	defer client.Close()

	tlsCfg := &tls.Config{
		ServerName: s.cfg.Host,
		MinVersion: tls.VersionTLS12,
	}
	if err := client.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("STARTTLS %s: %w", addr, err)
	}
	return sendWithClient(client, from, recipients, raw, auth)
}

// sendPlain connects without TLS (localhost relay or port 25).
func (s *EmailSender) sendPlain(addr, from string, recipients []string, raw []byte, auth smtp.Auth) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("SMTP dial %s: %w", addr, err)
	}
	defer client.Close()
	return sendWithClient(client, from, recipients, raw, auth)
}

// sendWithClient performs the SMTP DATA transaction on an already-connected client.
func sendWithClient(client *smtp.Client, from string, recipients []string, raw []byte, auth smtp.Auth) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("SMTP RCPT TO %s: %w", rcpt, err)
		}
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := wc.Write(raw); err != nil {
		_ = wc.Close()
		return fmt.Errorf("SMTP writing message: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("SMTP closing DATA writer: %w", err)
	}
	return client.Quit()
}

// collectRecipients returns the combined envelope recipients (To + Cc + Bcc),
// deduplicated.
func collectRecipients(msg EmailMessage) []string {
	seen := make(map[string]struct{})
	var all []string
	for _, list := range [][]string{msg.To, msg.Cc, msg.Bcc} {
		for _, addr := range list {
			if _, ok := seen[addr]; !ok {
				seen[addr] = struct{}{}
				all = append(all, addr)
			}
		}
	}
	return all
}

// buildRawMessage constructs the full MIME message bytes.
//
// Structure when there are attachments:
//
//	multipart/mixed
//	  multipart/alternative
//	    text/plain  (quoted-printable)
//	    text/html   (quoted-printable)
//	  attachment …
//
// Structure without attachments:
//
//	multipart/alternative
//	  text/plain
//	  text/html
func buildRawMessage(from string, msg EmailMessage) ([]byte, error) {
	var buf bytes.Buffer

	writeHeader(&buf, "From", from)
	writeHeader(&buf, "To", strings.Join(msg.To, ", "))
	if len(msg.Cc) > 0 {
		writeHeader(&buf, "Cc", strings.Join(msg.Cc, ", "))
	}
	writeHeader(&buf, "Subject", encodeHeader(msg.Subject))
	writeHeader(&buf, "MIME-Version", "1.0")
	writeHeader(&buf, "Date", time.Now().Format(time.RFC1123Z))

	if len(msg.Attachments) > 0 {
		if err := writeMixedBody(&buf, msg); err != nil {
			return nil, err
		}
	} else {
		if err := writeAlternativeBody(&buf, msg.BodyText, msg.BodyHTML); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// writeMixedBody writes a multipart/mixed body (alternative + attachments).
func writeMixedBody(buf *bytes.Buffer, msg EmailMessage) error {
	mw := multipart.NewWriter(buf)
	writeHeader(buf, "Content-Type", fmt.Sprintf(`multipart/mixed; boundary="%s"`, mw.Boundary()))
	buf.WriteString("\r\n")

	// Build the inner alternative part with a known boundary.
	altBoundary := "alt" + mw.Boundary()
	altBuf, err := buildAlternativeBytes(msg.BodyText, msg.BodyHTML, altBoundary)
	if err != nil {
		return err
	}
	altH := textproto.MIMEHeader{
		"Content-Type": {fmt.Sprintf(`multipart/alternative; boundary="%s"`, altBoundary)},
	}
	altPart, err := mw.CreatePart(altH)
	if err != nil {
		return fmt.Errorf("creating alternative part: %w", err)
	}
	if _, err := altPart.Write(altBuf); err != nil {
		return fmt.Errorf("writing alternative part: %w", err)
	}

	// Write attachments.
	for _, att := range msg.Attachments {
		if err := writeAttachment(mw, att); err != nil {
			return err
		}
	}
	return mw.Close()
}

// writeAlternativeBody writes a multipart/alternative body directly into buf.
func writeAlternativeBody(buf *bytes.Buffer, textBody, htmlBody string) error {
	aw := multipart.NewWriter(buf)
	writeHeader(buf, "Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, aw.Boundary()))
	buf.WriteString("\r\n")
	if err := writeTextPart(aw, textBody); err != nil {
		return err
	}
	if err := writeHTMLPart(aw, htmlBody); err != nil {
		return err
	}
	return aw.Close()
}

// buildAlternativeBytes builds the inner multipart/alternative content as bytes
// for embedding inside a multipart/mixed part. The boundary parameter ensures
// the boundary used in the body matches the one declared in the Content-Type header.
func buildAlternativeBytes(textBody, htmlBody, boundary string) ([]byte, error) {
	var buf bytes.Buffer
	aw := multipart.NewWriter(&buf)
	if err := aw.SetBoundary(boundary); err != nil {
		return nil, fmt.Errorf("setting alternative boundary: %w", err)
	}
	if err := writeTextPart(aw, textBody); err != nil {
		return nil, err
	}
	if err := writeHTMLPart(aw, htmlBody); err != nil {
		return nil, err
	}
	if err := aw.Close(); err != nil {
		return nil, fmt.Errorf("closing inner alternative: %w", err)
	}
	return buf.Bytes(), nil
}

func writeTextPart(w *multipart.Writer, body string) error {
	h := textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	}
	part, err := w.CreatePart(h)
	if err != nil {
		return fmt.Errorf("creating text/plain part: %w", err)
	}
	qpw := quotedprintable.NewWriter(part)
	if _, err := qpw.Write([]byte(body)); err != nil {
		return fmt.Errorf("writing text/plain body: %w", err)
	}
	return qpw.Close()
}

func writeHTMLPart(w *multipart.Writer, body string) error {
	h := textproto.MIMEHeader{
		"Content-Type":              {"text/html; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	}
	part, err := w.CreatePart(h)
	if err != nil {
		return fmt.Errorf("creating text/html part: %w", err)
	}
	qpw := quotedprintable.NewWriter(part)
	if _, err := qpw.Write([]byte(body)); err != nil {
		return fmt.Errorf("writing text/html body: %w", err)
	}
	return qpw.Close()
}

func writeAttachment(w *multipart.Writer, att Attachment) error {
	ct := att.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	h := textproto.MIMEHeader{
		"Content-Type":              {fmt.Sprintf(`%s; name="%s"`, ct, att.Filename)},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, att.Filename)},
	}
	part, err := w.CreatePart(h)
	if err != nil {
		return fmt.Errorf("creating attachment part %s: %w", att.Filename, err)
	}
	encoded := base64.StdEncoding.EncodeToString(att.Data)
	// Wrap at 76 chars per MIME spec.
	for len(encoded) > 76 {
		if _, err := fmt.Fprintf(part, "%s\r\n", encoded[:76]); err != nil {
			return fmt.Errorf("writing attachment data %s: %w", att.Filename, err)
		}
		encoded = encoded[76:]
	}
	if len(encoded) > 0 {
		if _, err := fmt.Fprintf(part, "%s\r\n", encoded); err != nil {
			return fmt.Errorf("writing attachment tail %s: %w", att.Filename, err)
		}
	}
	return nil
}

func writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

// encodeHeader encodes a header value as RFC 2047 UTF-8 base64 when it
// contains non-ASCII characters.
func encodeHeader(value string) string {
	for _, r := range value {
		if r > 127 {
			encoded := base64.StdEncoding.EncodeToString([]byte(value))
			return fmt.Sprintf("=?UTF-8?B?%s?=", encoded)
		}
	}
	return value
}
