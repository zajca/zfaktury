// Package eposchemas embeds the EPO (Finanční správa) XSD files that live
// next to this file, so tests and the XSD check use the same files as the
// reference copies without a second copy in the tree.
package eposchemas

import "embed"

// Files holds every XSD in this directory (DPFDP7, DPHDP3, DPHKH1, DPHSHV) at
// the root of the file system.
//
//go:embed *.xsd
var Files embed.FS
