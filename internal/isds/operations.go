package isds

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
)

// Privileges is the userPrivils bit set of a box user. The bit values are the
// documented ISDS ones (libdatovka isds_privileges).
type Privileges int64

// Box privilege bits.
const (
	PrivilegeReadNonPersonal Privileges = 0x1  // read messages not marked personal delivery
	PrivilegeReadAll         Privileges = 0x2  // read all messages
	PrivilegeCreateAndSend   Privileges = 0x4  // create and send messages
	PrivilegeViewInfo        Privileges = 0x8  // list messages and receipts
	PrivilegeSearchBoxes     Privileges = 0x10 // search for boxes
	PrivilegeAdministerBox   Privileges = 0x20 // administer the box
)

// CanCreateAndSend reports whether the user may create and send messages.
func (p Privileges) CanCreateAndSend() bool { return p&PrivilegeCreateAndSend != 0 }

// CanViewInfo reports whether the user may list sent messages and receipts.
func (p Privileges) CanViewInfo() bool { return p&PrivilegeViewInfo != 0 }

// CanReadMessages reports whether the user may read delivered messages, in
// which case logging in delivers them (doruceni prihlasenim).
func (p Privileges) CanReadMessages() bool {
	return p&(PrivilegeReadNonPersonal|PrivilegeReadAll) != 0
}

// OwnerInfo describes the box the login belongs to.
type OwnerInfo struct {
	BoxID     string
	BoxType   string
	FirstName string
	LastName  string
	FirmName  string
}

// UserInfo describes the logged-in user of the box.
type UserInfo struct {
	UserType   string
	Privileges Privileges
}

// DeliveryEvent is one entry of a message's event list.
type DeliveryEvent struct {
	Time        *time.Time
	Description string
}

// DeliveryInfo is the delivery evidence of a sent message.
type DeliveryInfo struct {
	MessageID      string
	Status         MessageStatus
	DeliveryTime   *time.Time
	AcceptanceTime *time.Time
	Events         []DeliveryEvent
}

// getOwnerInfoRequest and getUserInfoRequest carry the required but unused
// dbDummy element of tDummyInput.
type getOwnerInfoRequest struct {
	XMLName xml.Name `xml:"http://isds.czechpoint.cz/v20 GetOwnerInfoFromLogin2"`
	Dummy   string   `xml:"dbDummy"`
}

type getUserInfoRequest struct {
	XMLName xml.Name `xml:"http://isds.czechpoint.cz/v20 GetUserInfoFromLogin2"`
	Dummy   string   `xml:"dbDummy"`
}

type getDeliveryInfoRequest struct {
	XMLName xml.Name `xml:"http://isds.czechpoint.cz/v20 GetDeliveryInfo"`
	ID      string   `xml:"dmID"`
}

type getSignedDeliveryInfoRequest struct {
	XMLName xml.Name `xml:"http://isds.czechpoint.cz/v20 GetSignedDeliveryInfo"`
	ID      string   `xml:"dmID"`
}

type signedSentMessageDownloadRequest struct {
	XMLName xml.Name `xml:"http://isds.czechpoint.cz/v20 SignedSentMessageDownload"`
	ID      string   `xml:"dmID"`
}

type ownerInfoResponse struct {
	Owner struct {
		BoxID     string `xml:"dbID"`
		BoxType   string `xml:"dbType"`
		FirstName string `xml:"pnFirstName"`
		LastName  string `xml:"pnLastName"`
		FirmName  string `xml:"firmName"`
	} `xml:"dbOwnerInfo"`
	Status *dbStatus `xml:"dbStatus"`
}

type userInfoResponse struct {
	User *struct {
		UserType   string `xml:"userType"`
		Privileges int64  `xml:"userPrivils"`
	} `xml:"dbUserInfo"`
	Status *dbStatus `xml:"dbStatus"`
}

type createMessageResponse struct {
	ID     string    `xml:"dmID"`
	Status *dmStatus `xml:"dmStatus"`
}

type deliveryInfoResponse struct {
	Delivery *struct {
		ID             string `xml:"dmDm>dmID"`
		DeliveryTime   string `xml:"dmDeliveryTime"`
		AcceptanceTime string `xml:"dmAcceptanceTime"`
		Status         int    `xml:"dmMessageStatus"`
		Events         []struct {
			Time        string `xml:"dmEventTime"`
			Description string `xml:"dmEventDescr"`
		} `xml:"dmEvents>dmEvent"`
	} `xml:"dmDelivery"`
	Status *dmStatus `xml:"dmStatus"`
}

// signatureResponse is the answer of the operations returning a signed
// (ZFO) blob.
type signatureResponse struct {
	Signature string    `xml:"dmSignature"`
	Status    *dmStatus `xml:"dmStatus"`
}

// GetOwnerInfo calls GetOwnerInfoFromLogin2. It verifies the login and returns
// the box the login belongs to.
func (c *Client) GetOwnerInfo(ctx context.Context) (*OwnerInfo, error) {
	resp, err := call[ownerInfoResponse](ctx, c, pathAccessInfo, "GetOwnerInfoFromLogin2", getOwnerInfoRequest{})
	if err != nil {
		return nil, err
	}
	if err := checkDBStatus(resp.Status); err != nil {
		return nil, fmt.Errorf("GetOwnerInfoFromLogin2: %w", err)
	}
	return &OwnerInfo{
		BoxID:     resp.Owner.BoxID,
		BoxType:   resp.Owner.BoxType,
		FirstName: resp.Owner.FirstName,
		LastName:  resp.Owner.LastName,
		FirmName:  resp.Owner.FirmName,
	}, nil
}

// GetUserInfo calls GetUserInfoFromLogin2 and returns the user's privileges.
// A missing privilege element yields no privileges.
func (c *Client) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	resp, err := call[userInfoResponse](ctx, c, pathAccessInfo, "GetUserInfoFromLogin2", getUserInfoRequest{})
	if err != nil {
		return nil, err
	}
	if err := checkDBStatus(resp.Status); err != nil {
		return nil, fmt.Errorf("GetUserInfoFromLogin2: %w", err)
	}
	info := &UserInfo{}
	if resp.User != nil {
		info.UserType = resp.User.UserType
		info.Privileges = Privileges(resp.User.Privileges)
	}
	return info, nil
}

// CreateMessage validates and sends a data message and returns its dmID.
func (c *Client) CreateMessage(ctx context.Context, m MessageInput) (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	resp, err := call[createMessageResponse](ctx, c, pathMessageOperations, "CreateMessage", newCreateMessageRequest(m))
	if err != nil {
		return "", err
	}
	if err := checkDMStatus(resp.Status); err != nil {
		return "", fmt.Errorf("CreateMessage: %w", err)
	}
	if !messageIDPattern.MatchString(resp.ID) {
		return "", fmt.Errorf("CreateMessage: %w: missing or malformed dmID", ErrUnexpectedResponse)
	}
	return resp.ID, nil
}

// GetDeliveryInfo returns the delivery evidence (status, times, events) of a
// sent message.
func (c *Client) GetDeliveryInfo(ctx context.Context, dmID string) (*DeliveryInfo, error) {
	if !messageIDPattern.MatchString(dmID) {
		return nil, invalidInput("message id must be 1 to 20 digits")
	}
	resp, err := call[deliveryInfoResponse](ctx, c, pathMessageInfo, "GetDeliveryInfo", getDeliveryInfoRequest{ID: dmID})
	if err != nil {
		return nil, err
	}
	if err := checkDMStatus(resp.Status); err != nil {
		return nil, fmt.Errorf("GetDeliveryInfo: %w", err)
	}
	if resp.Delivery == nil {
		return nil, fmt.Errorf("GetDeliveryInfo: %w: missing dmDelivery", ErrUnexpectedResponse)
	}

	info := &DeliveryInfo{
		MessageID: resp.Delivery.ID,
		Status:    MessageStatus(resp.Delivery.Status),
	}
	if !info.Status.Valid() {
		return nil, fmt.Errorf("GetDeliveryInfo: %w: message status %d out of range", ErrUnexpectedResponse, resp.Delivery.Status)
	}
	if info.DeliveryTime, err = parseTime(resp.Delivery.DeliveryTime); err != nil {
		return nil, fmt.Errorf("GetDeliveryInfo: %w", err)
	}
	if info.AcceptanceTime, err = parseTime(resp.Delivery.AcceptanceTime); err != nil {
		return nil, fmt.Errorf("GetDeliveryInfo: %w", err)
	}
	for _, e := range resp.Delivery.Events {
		t, err := parseTime(e.Time)
		if err != nil {
			return nil, fmt.Errorf("GetDeliveryInfo: %w", err)
		}
		info.Events = append(info.Events, DeliveryEvent{Time: t, Description: e.Description})
	}
	return info, nil
}

// GetSignedDeliveryInfo returns the delivery receipt of a sent message as the
// signed ZFO blob.
func (c *Client) GetSignedDeliveryInfo(ctx context.Context, dmID string) ([]byte, error) {
	if !messageIDPattern.MatchString(dmID) {
		return nil, invalidInput("message id must be 1 to 20 digits")
	}
	resp, err := call[signatureResponse](ctx, c, pathMessageInfo, "GetSignedDeliveryInfo", getSignedDeliveryInfoRequest{ID: dmID})
	if err != nil {
		return nil, err
	}
	return signedBlob("GetSignedDeliveryInfo", resp)
}

// SignedSentMessageDownload returns the signed sent message as a ZFO blob.
func (c *Client) SignedSentMessageDownload(ctx context.Context, dmID string) ([]byte, error) {
	if !messageIDPattern.MatchString(dmID) {
		return nil, invalidInput("message id must be 1 to 20 digits")
	}
	resp, err := call[signatureResponse](ctx, c, pathMessageOperations, "SignedSentMessageDownload", signedSentMessageDownloadRequest{ID: dmID})
	if err != nil {
		return nil, err
	}
	return signedBlob("SignedSentMessageDownload", resp)
}

func signedBlob(op string, resp *signatureResponse) ([]byte, error) {
	if err := checkDMStatus(resp.Status); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	data, err := decodeBase64(resp.Signature)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%s: %w: empty dmSignature", op, ErrUnexpectedResponse)
	}
	return data, nil
}

// parseTime reads an xs:dateTime with a zone; an empty value (nil element) is
// no time.
func parseTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("%w: bad dateTime", ErrUnexpectedResponse)
	}
	return &t, nil
}
