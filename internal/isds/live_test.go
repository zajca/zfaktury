//go:build isdslive

// Live smoke test against the ISDS TEST environment (ws1.datovka-test.gov.cz).
// It is never run by CI or by default: it needs the build tag and the
// environment variables below, which hold the login of a test box created in
// the production portal (Nastaveni > Pro vyvojare > Testovaci schranky).
//
//	ZFAKTURY_ISDS_TEST_USERNAME  login name of the test box or delegated user
//	ZFAKTURY_ISDS_TEST_PASSWORD  its password
//
// Run: go test -tags isdslive -run TestLive ./internal/isds/
//
// The test only logs in and reads the owner and privileges; it sends nothing.
package isds

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLive_LoginAndPrivileges(t *testing.T) {
	username := os.Getenv("ZFAKTURY_ISDS_TEST_USERNAME")
	password := os.Getenv("ZFAKTURY_ISDS_TEST_PASSWORD")
	if username == "" || password == "" {
		t.Skip("ZFAKTURY_ISDS_TEST_USERNAME and ZFAKTURY_ISDS_TEST_PASSWORD are not set")
	}

	c, err := NewClient(EnvironmentTest, Credentials{Username: username, Password: password})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*DefaultTimeout)
	defer cancel()

	owner, err := c.GetOwnerInfo(ctx)
	if err != nil {
		t.Fatalf("GetOwnerInfo: %v", err)
	}
	if owner.BoxID == "" {
		t.Error("owner box id is empty")
	}

	time.Sleep(time.Second) // ISDS throttles clients that call in quick succession
	user, err := c.GetUserInfo(ctx)
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	t.Logf("box type %s, can create and send: %v, can read messages: %v",
		owner.BoxType, user.Privileges.CanCreateAndSend(), user.Privileges.CanReadMessages())
}
