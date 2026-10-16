//go:build !windows

package main

// ensureElevated 在非 Windows 平台始终返回 false。
func ensureElevated() bool { return false }

func msgBox(title, text string) {}
