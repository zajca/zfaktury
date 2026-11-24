package isds

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// statusErrorBody is a well-formed answer with a non-0000 status for an
// operation whose response element is name and whose status element is
// statusElem.
func statusErrorBody(name, statusElem string) []byte {
	return []byte(`<Envelope><Body><` + name + `><` + statusElem + `><` + statusElem + `Code>1234</` + statusElem + `Code><` +
		statusElem + `Message>Fake failure</` + statusElem + `Message></` + statusElem + `></` + name + `></Body></Envelope>`)
}

// TestEveryOperation_ErrorMapping runs the error cases against all five
// operations, so none of them depends on the shared plumbing by assumption.
func TestEveryOperation_ErrorMapping(t *testing.T) {
	ops := []struct {
		name       string
		response   string
		statusElem string
		run        func(c *Client) error
	}{
		{"GetOwnerInfo", "GetOwnerInfoFromLogin2Response", "dbStatus", func(c *Client) error {
			_, err := c.GetOwnerInfo(context.Background())
			return err
		}},
		{"GetUserInfo", "GetUserInfoFromLogin2Response", "dbStatus", func(c *Client) error {
			_, err := c.GetUserInfo(context.Background())
			return err
		}},
		{"CreateMessage", "CreateMessageResponse", "dmStatus", func(c *Client) error {
			_, err := c.CreateMessage(context.Background(), validMessage())
			return err
		}},
		{"GetDeliveryInfo", "GetDeliveryInfoResponse", "dmStatus", func(c *Client) error {
			_, err := c.GetDeliveryInfo(context.Background(), "1234567890")
			return err
		}},
		{"GetSignedDeliveryInfo", "GetSignedDeliveryInfoResponse", "dmStatus", func(c *Client) error {
			_, err := c.GetSignedDeliveryInfo(context.Background(), "1234567890")
			return err
		}},
		{"SignedSentMessageDownload", "SignedSentMessageDownloadResponse", "dmStatus", func(c *Client) error {
			_, err := c.SignedSentMessageDownload(context.Background(), "1234567890")
			return err
		}},
	}

	for _, op := range ops {
		cases := []struct {
			name   string
			status int
			body   []byte
			want   error
		}{
			{"401", http.StatusUnauthorized, nil, ErrUnauthorized},
			{"500", http.StatusInternalServerError, []byte("boom"), ErrISDSUnavailable},
			{"fault", http.StatusInternalServerError, fixture(t, "fault.xml"), ErrSOAPFault},
			{"html on 200", http.StatusOK, fixture(t, "maintenance.html"), ErrUnexpectedResponse},
			{"status not 0000", http.StatusOK, statusErrorBody(op.response, op.statusElem), ErrISDSRejected},
		}
		for _, tc := range cases {
			t.Run(op.name+"/"+tc.name, func(t *testing.T) {
				srv, rec := newFake(t, tc.status, tc.body)
				err := op.run(newTestClient(t, srv))
				wantIs(t, err, tc.want)
				if err != nil && (strings.Contains(err.Error(), testPassword) || strings.Contains(err.Error(), testUsername)) {
					t.Errorf("error leaks the login: %v", err)
				}
				if rec.count() != 1 {
					t.Errorf("requests = %d, want 1", rec.count())
				}
			})
		}
	}
}
