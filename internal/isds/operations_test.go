package isds

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func validMessage() MessageInput {
	return MessageInput{
		RecipientBoxID:   "abc1234",
		Annotation:       "Fake filing",
		ToHands:          "Fake Hands",
		SenderRefNumber:  "REF-1",
		SenderIdent:      "IDENT-1",
		PersonalDelivery: true,
		Files: []File{
			{Name: "filing.xml", MIMEType: MIMETypeXML, MetaType: FileMain, Content: []byte("<a/>")},
			{Name: "note & more.pdf", MIMEType: "application/pdf", Content: []byte("%PDF")},
		},
	}
}

func TestGetOwnerInfo(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, fixture(t, "get_owner_info_response.xml"))
	info, err := newTestClient(t, srv).GetOwnerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetOwnerInfo: %v", err)
	}
	assertRequest(t, rec, "/DS/DsManage", "get_owner_info_request.xml")
	want := OwnerInfo{BoxID: "abc1234", BoxType: "FO", GivenNames: "Jan Karel", LastName: "Testovaci"}
	if *info != want {
		t.Errorf("info = %+v, want %+v", *info, want)
	}
}

func TestGetUserInfo(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, fixture(t, "get_user_info_response.xml"))
	info, err := newTestClient(t, srv).GetUserInfo(context.Background())
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	assertRequest(t, rec, "/DS/DsManage", "get_user_info_request.xml")
	if info.UserType != "OFFICIAL" || info.Privileges != 12 {
		t.Errorf("info = %+v", info)
	}
	if !info.Privileges.CanCreateAndSend() || !info.Privileges.CanViewInfo() || info.Privileges.CanReadMessages() {
		t.Errorf("privileges 12 decoded wrongly: send=%v view=%v read=%v",
			info.Privileges.CanCreateAndSend(), info.Privileges.CanViewInfo(), info.Privileges.CanReadMessages())
	}
}

func TestPrivileges(t *testing.T) {
	tests := []struct {
		p                  Privileges
		send, view, readAn bool
	}{
		{0, false, false, false},
		{PrivilegeCreateAndSend, true, false, false},
		{PrivilegeViewInfo, false, true, false},
		{PrivilegeReadNonPersonal, false, false, true},
		{PrivilegeReadAll | PrivilegeCreateAndSend, true, false, true},
		{0xFF, true, true, true},
	}
	for _, tt := range tests {
		if tt.p.CanCreateAndSend() != tt.send || tt.p.CanViewInfo() != tt.view || tt.p.CanReadMessages() != tt.readAn {
			t.Errorf("privileges %#x: send=%v view=%v read=%v", int64(tt.p),
				tt.p.CanCreateAndSend(), tt.p.CanViewInfo(), tt.p.CanReadMessages())
		}
	}
}

func TestCreateMessage(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, fixture(t, "create_message_response_ok.xml"))
	id, err := newTestClient(t, srv).CreateMessage(context.Background(), validMessage())
	if err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	if id != "1234567890" {
		t.Errorf("dmID = %q", id)
	}
	assertRequest(t, rec, "/DS/dz", "create_message_request.xml")
}

func TestCreateMessage_MalformedAnswers(t *testing.T) {
	for name, body := range map[string]string{
		"missing dmID":     `<Envelope><Body><CreateMessageResponse><dmStatus><dmStatusCode>0000</dmStatusCode><dmStatusMessage>ok</dmStatusMessage></dmStatus></CreateMessageResponse></Body></Envelope>`,
		"missing dmStatus": `<Envelope><Body><CreateMessageResponse><dmID>1</dmID></CreateMessageResponse></Body></Envelope>`,
		"non-numeric dmID": `<Envelope><Body><CreateMessageResponse><dmID>x1</dmID><dmStatus><dmStatusCode>0000</dmStatusCode><dmStatusMessage>ok</dmStatusMessage></dmStatus></CreateMessageResponse></Body></Envelope>`,
	} {
		t.Run(name, func(t *testing.T) {
			srv, _ := newFake(t, http.StatusOK, []byte(body))
			_, err := newTestClient(t, srv).CreateMessage(context.Background(), validMessage())
			wantIs(t, err, ErrUnexpectedResponse)
		})
	}
}

func TestGetDeliveryInfo(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, fixture(t, "get_delivery_info_response.xml"))
	info, err := newTestClient(t, srv).GetDeliveryInfo(context.Background(), "1234567890")
	if err != nil {
		t.Fatalf("GetDeliveryInfo: %v", err)
	}
	assertRequest(t, rec, "/DS/dx", "get_delivery_info_request.xml")

	if info.MessageID != "1234567890" || info.Status != StatusDelivered || !info.Status.Delivered() {
		t.Errorf("info = %+v", info)
	}
	wantTime := time.Date(2026, 11, 24, 10, 15, 30, 123_000_000, time.FixedZone("", 3600))
	if info.DeliveryTime == nil || !info.DeliveryTime.Equal(wantTime) {
		t.Errorf("DeliveryTime = %v, want %v", info.DeliveryTime, wantTime)
	}
	if info.AcceptanceTime != nil {
		t.Errorf("AcceptanceTime = %v, want nil for an nil element", info.AcceptanceTime)
	}
	if len(info.Events) != 2 || info.Events[0].Description != "EV1: message delivered" || info.Events[1].Time == nil {
		t.Errorf("events = %+v", info.Events)
	}
}

func TestGetDeliveryInfo_BadStatusOrTime(t *testing.T) {
	const tmpl = `<Envelope><Body><GetDeliveryInfoResponse><dmDelivery><dmDm><dmID>1</dmID></dmDm><dmDeliveryTime>%s</dmDeliveryTime><dmMessageStatus>%s</dmMessageStatus></dmDelivery><dmStatus><dmStatusCode>0000</dmStatusCode><dmStatusMessage>ok</dmStatusMessage></dmStatus></GetDeliveryInfoResponse></Body></Envelope>`
	for name, args := range map[string][2]string{
		"status out of range": {"2026-11-24T10:00:00+01:00", "12"},
		"bad time":            {"yesterday", "4"},
	} {
		t.Run(name, func(t *testing.T) {
			body := strings.Replace(strings.Replace(tmpl, "%s", args[0], 1), "%s", args[1], 1)
			srv, _ := newFake(t, http.StatusOK, []byte(body))
			_, err := newTestClient(t, srv).GetDeliveryInfo(context.Background(), "1")
			wantIs(t, err, ErrUnexpectedResponse)
		})
	}
}

func TestGetDeliveryInfo_MissingDelivery(t *testing.T) {
	const body = `<Envelope><Body><GetDeliveryInfoResponse><dmStatus><dmStatusCode>0000</dmStatusCode><dmStatusMessage>ok</dmStatusMessage></dmStatus></GetDeliveryInfoResponse></Body></Envelope>`
	srv, _ := newFake(t, http.StatusOK, []byte(body))
	_, err := newTestClient(t, srv).GetDeliveryInfo(context.Background(), "1")
	wantIs(t, err, ErrUnexpectedResponse)
}

func TestGetSignedDeliveryInfo(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, fixture(t, "get_signed_delivery_info_response.xml"))
	blob, err := newTestClient(t, srv).GetSignedDeliveryInfo(context.Background(), "1234567890")
	if err != nil {
		t.Fatalf("GetSignedDeliveryInfo: %v", err)
	}
	assertRequest(t, rec, "/DS/dx", "get_signed_delivery_info_request.xml")
	if string(blob) != "fake-zfo" {
		t.Errorf("blob = %q", blob)
	}
}

func TestSignedSentMessageDownload(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, fixture(t, "signed_sent_message_download_response.xml"))
	blob, err := newTestClient(t, srv).SignedSentMessageDownload(context.Background(), "1234567890")
	if err != nil {
		t.Fatalf("SignedSentMessageDownload: %v", err)
	}
	assertRequest(t, rec, "/DS/dz", "signed_sent_message_download_request.xml")
	if string(blob) != "fake-zfo" {
		t.Errorf("blob = %q", blob)
	}
}

func TestSignedBlob_BadAnswers(t *testing.T) {
	const tmpl = `<Envelope><Body><SignedSentMessageDownloadResponse>%s<dmStatus><dmStatusCode>%s</dmStatusCode><dmStatusMessage>m</dmStatusMessage></dmStatus></SignedSentMessageDownloadResponse></Body></Envelope>`
	tests := []struct {
		name, sig, code string
		want            error
	}{
		{"invalid base64", "<dmSignature>***</dmSignature>", "0000", ErrUnexpectedResponse},
		{"missing signature", "", "0000", ErrUnexpectedResponse},
		{"status error", "", "0009", ErrISDSRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.Replace(strings.Replace(tmpl, "%s", tt.sig, 1), "%s", tt.code, 1)
			srv, _ := newFake(t, http.StatusOK, []byte(body))
			_, err := newTestClient(t, srv).SignedSentMessageDownload(context.Background(), "1")
			wantIs(t, err, tt.want)
		})
	}
}

func TestMessageIDsAreValidatedBeforeSending(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, nil)
	c := newTestClient(t, srv)
	for _, id := range []string{"", "12a", "1 2", "123456789012345678901", "<x/>"} {
		if _, err := c.GetDeliveryInfo(context.Background(), id); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("GetDeliveryInfo(%q) err = %v", id, err)
		}
		if _, err := c.GetSignedDeliveryInfo(context.Background(), id); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("GetSignedDeliveryInfo(%q) err = %v", id, err)
		}
		if _, err := c.SignedSentMessageDownload(context.Background(), id); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("SignedSentMessageDownload(%q) err = %v", id, err)
		}
	}
	if rec.count() != 0 {
		t.Errorf("%d request(s) sent for invalid ids", rec.count())
	}
}

func TestCreateMessage_Validation(t *testing.T) {
	runes := func(n int) string { return strings.Repeat("ž", n) }
	tests := []struct {
		name   string
		mutate func(m *MessageInput)
	}{
		{"no recipient", func(m *MessageInput) { m.RecipientBoxID = "" }},
		{"short recipient", func(m *MessageInput) { m.RecipientBoxID = "abc123" }},
		{"recipient with symbols", func(m *MessageInput) { m.RecipientBoxID = "abc-234" }},
		{"annotation too long", func(m *MessageInput) { m.Annotation = runes(256) }},
		{"toHands too long", func(m *MessageInput) { m.ToHands = runes(31) }},
		{"sender ref too long", func(m *MessageInput) { m.SenderRefNumber = runes(51) }},
		{"sender ident too long", func(m *MessageInput) { m.SenderIdent = runes(51) }},
		{"control character in annotation", func(m *MessageInput) { m.Annotation = "a\x00b" }},
		{"no files", func(m *MessageInput) { m.Files = nil }},
		{"too many files", func(m *MessageInput) {
			m.Files = []File{{Name: "a.xml", MIMEType: MIMETypeXML, MetaType: FileMain, Content: []byte("<a/>")}}
			for i := 0; i < MaxFiles; i++ {
				m.Files = append(m.Files, File{Name: "e.pdf", MIMEType: "application/pdf", Content: []byte("x")})
			}
		}},
		{"first file not main", func(m *MessageInput) { m.Files[0].MetaType = FileEnclosure }},
		{"first file without meta type", func(m *MessageInput) { m.Files[0].MetaType = "" }},
		{"unknown meta type", func(m *MessageInput) { m.Files[1].MetaType = "attachment" }},
		{"file name too long", func(m *MessageInput) { m.Files[1].Name = runes(256) }},
		{"file without name", func(m *MessageInput) { m.Files[1].Name = "" }},
		{"file without MIME type", func(m *MessageInput) { m.Files[1].MIMEType = "" }},
		{"empty file", func(m *MessageInput) { m.Files[1].Content = nil }},
		{"xml with wrong MIME type", func(m *MessageInput) { m.Files[0].MIMEType = "text/xml" }},
		{"upper-case xml with wrong MIME type", func(m *MessageInput) { m.Files[0].Name = "FILING.XML"; m.Files[0].MIMEType = "text/plain" }},
		{"over 20 MB", func(m *MessageInput) {
			m.Files[1].Content = make([]byte, MaxMessageBytes)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, rec := newFake(t, http.StatusOK, fixture(t, "create_message_response_ok.xml"))
			m := validMessage()
			tt.mutate(&m)

			_, err := newTestClient(t, srv).CreateMessage(context.Background(), m)
			wantIs(t, err, ErrInvalidInput)
			if rec.count() != 0 {
				t.Errorf("%d request(s) sent for invalid input", rec.count())
			}
		})
	}
}

func TestCreateMessage_LimitsAreInclusive(t *testing.T) {
	runes := func(n int) string { return strings.Repeat("ž", n) }
	m := validMessage()
	m.Annotation = runes(MaxAnnotationRunes)
	m.ToHands = runes(MaxToHandsRunes)
	m.SenderRefNumber = runes(MaxSenderRefRunes)
	m.SenderIdent = runes(MaxSenderIdentRunes)
	m.Files[1].Name = runes(MaxFileDescrRunes-4) + ".pdf"
	m.Files[1].Content = make([]byte, MaxMessageBytes-int64(len(m.Files[0].Content)))
	if err := m.Validate(); err != nil {
		t.Fatalf("limits are inclusive but Validate failed: %v", err)
	}

	m.Files = m.Files[:1]
	for len(m.Files) < MaxFiles {
		m.Files = append(m.Files, File{Name: "e.pdf", MIMEType: "application/pdf", Content: []byte("x")})
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("exactly %d files must be accepted: %v", MaxFiles, err)
	}
}

func TestCreateMessage_UnsetFieldsAreNil(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, fixture(t, "create_message_response_ok.xml"))
	m := MessageInput{
		RecipientBoxID: "abc1234",
		Files:          []File{{Name: "f.xml", MIMEType: MIMETypeXML, MetaType: FileMain, Content: []byte("<a/>")}},
	}
	if _, err := newTestClient(t, srv).CreateMessage(context.Background(), m); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	body := string(rec.last(t).body)
	for _, tag := range []string{"dmToHands", "dmAnnotation", "dmSenderRefNumber", "dmSenderIdent"} {
		if want := "<" + tag + ` xsi:nil="true"></` + tag + ">"; !strings.Contains(body, want) {
			t.Errorf("unset %s is not sent as a nil element in %s", tag, body)
		}
	}
	if !strings.Contains(body, "<dmPersonalDelivery>false</dmPersonalDelivery>") {
		t.Errorf("dmPersonalDelivery must be an explicit false in %s", body)
	}
}

func TestMessageStatus(t *testing.T) {
	want := map[MessageStatus]bool{
		0: false, 1: false, 2: false, 3: false,
		4: true, 5: true, 6: true, 7: true,
		8: false, 9: true, 10: false, -1: false,
	}
	for s, delivered := range want {
		if s.Delivered() != delivered {
			t.Errorf("MessageStatus(%d).Delivered() = %v, want %v", s, s.Delivered(), delivered)
		}
	}
	for s := MessageStatus(1); s <= 9; s++ {
		if !s.Valid() {
			t.Errorf("status %d must be valid", s)
		}
	}
	if MessageStatus(0).Valid() || MessageStatus(10).Valid() {
		t.Error("0 and 10 must be invalid")
	}
	if StatusSubmitted != 1 || StatusStamped != 2 || StatusRejectedMalicious != 3 || StatusDelivered != 4 ||
		StatusDeliveredByFiction != 5 || StatusDeliveredByLogin != 6 || StatusRead != 7 ||
		StatusUndeliverable != 8 || StatusDeleted != 9 {
		t.Error("status constants changed")
	}
}
