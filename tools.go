//go:build tools

package tools

// These imports ensure that Go module dependencies are tracked even before
// they are used in production code. Remove individual imports as actual
// usage is added to the codebase.
import (
	_ "github.com/dundee/qrpay"
	_ "github.com/johnfercher/maroto/v2"
)
