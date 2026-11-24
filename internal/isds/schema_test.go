package isds

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRequestsValidateAgainstXSD validates every operation element the client
// sends against the committed ISDS schemas with xmllint, so the request
// shapes are checked facts and not a reading of the WSDL. Skipped when
// xmllint is not installed.
func TestRequestsValidateAgainstXSD(t *testing.T) {
	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		t.Skip("xmllint not installed")
	}
	wsdlDir := filepath.Join("..", "..", "docs", "isds", "wsdl")

	tests := []struct {
		name   string
		schema string
		op     any
	}{
		{"CreateMessage", "dmBaseTypes.xsd", newCreateMessageRequest(validMessage())},
		{"CreateMessage minimal", "dmBaseTypes.xsd", newCreateMessageRequest(MessageInput{
			RecipientBoxID: "abc1234",
			Files:          []File{{Name: "f.xml", MIMEType: MIMETypeXML, MetaType: FileMain, Content: []byte("<a/>")}},
		})},
		{"GetDeliveryInfo", "dmBaseTypes.xsd", getDeliveryInfoRequest{ID: "1234567890"}},
		{"GetSignedDeliveryInfo", "dmBaseTypes.xsd", getSignedDeliveryInfoRequest{ID: "1234567890"}},
		{"SignedSentMessageDownload", "dmBaseTypes.xsd", signedSentMessageDownloadRequest{ID: "1234567890"}},
		{"GetOwnerInfoFromLogin2", "dbTypes.xsd", getOwnerInfoRequest{}},
		{"GetUserInfoFromLogin2", "dbTypes.xsd", getUserInfoRequest{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := xml.Marshal(tt.op)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			file := filepath.Join(t.TempDir(), "request.xml")
			if err := os.WriteFile(file, doc, 0o600); err != nil {
				t.Fatalf("writing request: %v", err)
			}
			out, err := exec.Command(xmllint, "--noout", "--schema", filepath.Join(wsdlDir, tt.schema), file).CombinedOutput()
			if err != nil {
				t.Errorf("request does not validate: %v\n%s\nrequest: %s", err, out, doc)
			}
		})
	}
}
