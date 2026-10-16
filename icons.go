package main

import "embed"

//go:embed assets/icon_on.ico assets/icon_off.ico assets/icon_unknown.ico
var iconFS embed.FS

var (
	iconOn      = mustReadIcon("assets/icon_on.ico")
	iconOff     = mustReadIcon("assets/icon_off.ico")
	iconUnknown = mustReadIcon("assets/icon_unknown.ico")
)

func mustReadIcon(name string) []byte {
	b, err := iconFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return b
}
