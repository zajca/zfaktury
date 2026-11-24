package domain

import "time"

// FilingKind names a generated filing that can be sent through a data box.
type FilingKind string

// Filing kinds, as used in the API path and the filing_submissions table.
const (
	FilingKindVATReturn        FilingKind = "vat-return"
	FilingKindControlStatement FilingKind = "control-statement"
	FilingKindVIESSummary      FilingKind = "vies-summary"
	FilingKindIncomeTax        FilingKind = "income-tax-return"
	FilingKindSocialInsurance  FilingKind = "social-insurance"
)

// FilingChannel is the route a filing was sent through.
type FilingChannel string

// FilingChannelISDS is the data-box (ISDS) channel.
const FilingChannelISDS FilingChannel = "isds"

// FilingSubmissionStatus is the life-cycle state of a send attempt.
type FilingSubmissionStatus string

// Submission states. Only Sending, Unknown, Sent and Delivered block a second
// send of the same bytes to the same environment.
const (
	// SubmissionSending means the request is about to leave or is in flight.
	SubmissionSending FilingSubmissionStatus = "sending"
	// SubmissionUnknown means CreateMessage ended in a transport failure, so
	// the message may or may not have been accepted.
	SubmissionUnknown FilingSubmissionStatus = "unknown"
	// SubmissionSent means ISDS accepted the message and delivery is not
	// confirmed yet.
	SubmissionSent FilingSubmissionStatus = "sent"
	// SubmissionDelivered means dmMessageStatus is 4 or higher except 8.
	SubmissionDelivered FilingSubmissionStatus = "delivered"
	// SubmissionUndeliverable means dmMessageStatus is 8.
	SubmissionUndeliverable FilingSubmissionStatus = "undeliverable"
	// SubmissionFailed means ISDS definitively refused the message.
	SubmissionFailed FilingSubmissionStatus = "failed"
)

// FilingSubmission is one attempt to send a filing through a data box. The XML
// bytes are not stored; XMLSHA256 identifies them.
type FilingSubmission struct {
	ID             int64
	DocumentKind   FilingKind
	DocumentID     int64
	Channel        FilingChannel
	Environment    string
	RecipientBoxID string
	RecipientName  string
	DmID           string
	DmAnnotation   string
	FileName       string
	XMLSHA256      string
	Status         FilingSubmissionStatus
	// DmMessageStatus is the last dmMessageStatus received as is, nil until a
	// delivery query ran.
	DmMessageStatus *int
	SentAt          *time.Time
	DeliveredAt     *time.Time
	// HasReceipt reports whether the signed message (ZFO) is stored; the blob
	// itself is read separately.
	HasReceipt bool
	ErrorText  string
	CreatedAt  time.Time
}
