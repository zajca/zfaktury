// Package csszschemas embeds the ČSSZ e-Podání XSD files that live next to
// this file, so the runtime XSD check uses the same files as the reference
// copies without a second copy in the tree.
package csszschemas

import "embed"

// Files holds every XSD in this directory (OSVC24.xsd, OSVC25.xsd and the
// imported baseTypes2.xsd) at the root of the file system.
//
//go:embed *.xsd
var Files embed.FS
