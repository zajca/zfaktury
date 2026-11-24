package isds

import (
	"encoding/base64"
	"encoding/xml"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ISDS limits for CreateMessage (WS_manipulace_s_datovymi_zpravami.pdf s.2.1
// and the length restrictions in dmBaseTypes.xsd).
const (
	MaxAnnotationRunes        = 255
	MaxToHandsRunes           = 30
	MaxSenderRefRunes         = 50
	MaxSenderIdentRunes       = 50
	MaxFileDescrRunes         = 255
	MaxFiles                  = 100
	MaxMessageBytes     int64 = 20 << 20

	// MIMETypeXML is the MIME type every .xml file must be sent with.
	MIMETypeXML = "application/xml"
)

var (
	boxIDPattern     = regexp.MustCompile(`^[A-Za-z0-9]{7}$`)
	messageIDPattern = regexp.MustCompile(`^[0-9]{1,20}$`)
)

// FileMetaType is the dmFileMetaType of a message file.
type FileMetaType string

// The dmFileMetaType values of dmBaseTypes.xsd.
const (
	FileMain      FileMetaType = "main"
	FileEnclosure FileMetaType = "enclosure"
	FileSignature FileMetaType = "signature"
	FileMeta      FileMetaType = "meta"
)

func (t FileMetaType) valid() bool {
	switch t {
	case FileMain, FileEnclosure, FileSignature, FileMeta:
		return true
	}
	return false
}

// File is one document of a data message. Content is sent base64-encoded.
type File struct {
	// Name is the dmFileDescr (file name), at most 255 characters.
	Name string
	// MIMEType is the dmMimeType, for example "application/xml".
	MIMEType string
	// MetaType is the dmFileMetaType. The first file must be FileMain; an
	// empty value on a later file means FileEnclosure.
	MetaType FileMetaType
	Content  []byte
}

// MessageInput is a data message to create.
type MessageInput struct {
	// RecipientBoxID is the 7-character id of the recipient box (required).
	RecipientBoxID string
	// Annotation is the subject (dmAnnotation), at most 255 characters.
	Annotation string
	// ToHands is dmToHands, at most 30 characters.
	ToHands string
	// SenderRefNumber is dmSenderRefNumber, at most 50 characters.
	SenderRefNumber string
	// SenderIdent is dmSenderIdent, at most 50 characters.
	SenderIdent string
	// PersonalDelivery sets dmPersonalDelivery.
	PersonalDelivery bool
	Files            []File
}

// Validate checks the message against the ISDS limits. CreateMessage calls it
// before sending anything.
func (m MessageInput) Validate() error {
	if !boxIDPattern.MatchString(m.RecipientBoxID) {
		return invalidInput("recipient box id must be 7 letters or digits")
	}
	for _, f := range []struct {
		name  string
		value string
		max   int
	}{
		{"dmAnnotation", m.Annotation, MaxAnnotationRunes},
		{"dmToHands", m.ToHands, MaxToHandsRunes},
		{"dmSenderRefNumber", m.SenderRefNumber, MaxSenderRefRunes},
		{"dmSenderIdent", m.SenderIdent, MaxSenderIdentRunes},
	} {
		if err := checkText(f.name, f.value, f.max); err != nil {
			return err
		}
	}

	if len(m.Files) == 0 {
		return invalidInput("at least one file is required")
	}
	if len(m.Files) > MaxFiles {
		return invalidInput("at most %d files are allowed", MaxFiles)
	}

	var total int64
	for i, f := range m.Files {
		if err := validateFile(i, f); err != nil {
			return err
		}
		total += int64(len(f.Content))
	}
	if total > MaxMessageBytes {
		return invalidInput("files exceed the %d byte message limit", MaxMessageBytes)
	}
	return nil
}

func validateFile(i int, f File) error {
	wantMain := i == 0
	meta := f.MetaType
	if meta == "" && !wantMain {
		meta = FileEnclosure
	}
	switch {
	case wantMain && meta != FileMain:
		return invalidInput("the first file must have meta type %q", FileMain)
	case !meta.valid():
		return invalidInput("file %d has an unknown meta type", i)
	case f.Name == "":
		return invalidInput("file %d has no name", i)
	case f.MIMEType == "":
		return invalidInput("file %d has no MIME type", i)
	case len(f.Content) == 0:
		return invalidInput("file %d is empty", i)
	}
	if err := checkText("dmFileDescr", f.Name, MaxFileDescrRunes); err != nil {
		return err
	}
	if err := checkText("dmMimeType", f.MIMEType, MaxFileDescrRunes); err != nil {
		return err
	}
	if strings.EqualFold(path.Ext(f.Name), ".xml") && f.MIMEType != MIMETypeXML {
		return invalidInput("file %d is XML and must have MIME type %s", i, MIMETypeXML)
	}
	return nil
}

// checkText enforces the maximum length and rejects characters XML 1.0 cannot
// carry, which encoding/xml would silently replace.
func checkText(field, value string, maxRunes int) error {
	if utf8.RuneCountInString(value) > maxRunes {
		return invalidInput("%s is longer than %d characters", field, maxRunes)
	}
	for _, r := range value {
		if !isXMLChar(r) {
			return invalidInput("%s contains a character that XML cannot carry", field)
		}
	}
	return nil
}

func isXMLChar(r rune) bool {
	return r == '\t' || r == '\n' || r == '\r' ||
		(r >= 0x20 && r <= 0xD7FF) ||
		(r >= 0xE000 && r <= 0xFFFD && r != 0xFFFE && r != 0xFFFF) ||
		(r >= 0x10000 && r <= 0x10FFFF)
}

const xsiNS = "http://www.w3.org/2001/XMLSchema-instance"

// nillable is an envelope element that is required by the schema but may be
// nil: an unset value is sent as xsi:nil="true".
type nillable struct {
	Nil  string `xml:"xsi:nil,attr,omitempty"`
	Text string `xml:",chardata"`
}

func optional(s string) nillable {
	if s == "" {
		return nillable{Nil: "true"}
	}
	return nillable{Text: s}
}

// createMessageRequest is the CreateMessage element. dmEnvelope carries every
// element of gMessageEnvelopeSub in the schema's order, as the schema makes
// them all required; dmOVM and dmPublishOwnID are optional and left out.
type createMessageRequest struct {
	XMLName  xml.Name `xml:"http://isds.czechpoint.cz/v20 CreateMessage"`
	XSI      string   `xml:"xmlns:xsi,attr"`
	Envelope struct {
		SenderOrgUnit       nillable `xml:"dmSenderOrgUnit"`
		SenderOrgUnitNum    nillable `xml:"dmSenderOrgUnitNum"`
		RecipientBoxID      string   `xml:"dbIDRecipient"`
		RecipientOrgUnit    nillable `xml:"dmRecipientOrgUnit"`
		RecipientOrgUnitNum nillable `xml:"dmRecipientOrgUnitNum"`
		ToHands             nillable `xml:"dmToHands"`
		Annotation          nillable `xml:"dmAnnotation"`
		RecipientRefNumber  nillable `xml:"dmRecipientRefNumber"`
		SenderRefNumber     nillable `xml:"dmSenderRefNumber"`
		RecipientIdent      nillable `xml:"dmRecipientIdent"`
		SenderIdent         nillable `xml:"dmSenderIdent"`
		LegalTitleLaw       nillable `xml:"dmLegalTitleLaw"`
		LegalTitleYear      nillable `xml:"dmLegalTitleYear"`
		LegalTitleSect      nillable `xml:"dmLegalTitleSect"`
		LegalTitlePar       nillable `xml:"dmLegalTitlePar"`
		LegalTitlePoint     nillable `xml:"dmLegalTitlePoint"`
		PersonalDelivery    nillable `xml:"dmPersonalDelivery"`
		AllowSubstDelivery  nillable `xml:"dmAllowSubstDelivery"`
	} `xml:"dmEnvelope"`
	Files struct {
		File []requestFile `xml:"dmFile"`
	} `xml:"dmFiles"`
}

type requestFile struct {
	MIMEType string `xml:"dmMimeType,attr"`
	MetaType string `xml:"dmFileMetaType,attr"`
	Descr    string `xml:"dmFileDescr,attr"`
	Content  string `xml:"dmEncodedContent"`
}

func newCreateMessageRequest(m MessageInput) createMessageRequest {
	r := createMessageRequest{XSI: xsiNS}
	e := &r.Envelope
	nilValue := optional("")
	e.SenderOrgUnit, e.SenderOrgUnitNum = nilValue, nilValue
	e.RecipientBoxID = m.RecipientBoxID
	e.RecipientOrgUnit, e.RecipientOrgUnitNum = nilValue, nilValue
	e.ToHands = optional(m.ToHands)
	e.Annotation = optional(m.Annotation)
	e.RecipientRefNumber = nilValue
	e.SenderRefNumber = optional(m.SenderRefNumber)
	e.RecipientIdent = nilValue
	e.SenderIdent = optional(m.SenderIdent)
	e.LegalTitleLaw, e.LegalTitleYear = nilValue, nilValue
	e.LegalTitleSect, e.LegalTitlePar, e.LegalTitlePoint = nilValue, nilValue, nilValue
	e.PersonalDelivery = optional(strconv.FormatBool(m.PersonalDelivery))
	e.AllowSubstDelivery = nilValue
	for i, f := range m.Files {
		meta := f.MetaType
		if meta == "" && i > 0 {
			meta = FileEnclosure
		}
		r.Files.File = append(r.Files.File, requestFile{
			MIMEType: f.MIMEType,
			MetaType: string(meta),
			Descr:    f.Name,
			Content:  base64.StdEncoding.EncodeToString(f.Content),
		})
	}
	return r
}

// MessageStatus is dmMessageStatus, the life-cycle state of a data message.
type MessageStatus int

// The dmMessageStatus values 1-9.
const (
	StatusSubmitted          MessageStatus = 1 // podana
	StatusStamped            MessageStatus = 2 // opatrena casovym razitkem
	StatusRejectedMalicious  MessageStatus = 3 // obsahuje virus
	StatusDelivered          MessageStatus = 4 // dodana do schranky
	StatusDeliveredByFiction MessageStatus = 5 // dorucena fikci
	StatusDeliveredByLogin   MessageStatus = 6 // dorucena prihlasenim
	StatusRead               MessageStatus = 7 // prectena
	StatusUndeliverable      MessageStatus = 8 // nedorucitelna
	StatusDeleted            MessageStatus = 9 // obsah smazan
)

// Valid reports whether s is one of the documented values 1-9.
func (s MessageStatus) Valid() bool { return s >= StatusSubmitted && s <= StatusDeleted }

// Delivered reports whether the message reached the recipient's box: status
// 4 or higher, except 8 (undeliverable).
func (s MessageStatus) Delivered() bool {
	return s.Valid() && s >= StatusDelivered && s != StatusUndeliverable
}
