//go:build !windows

package main

var errNoAutostart = errUnsupportedOS

func autostartEnabled() bool { return false }

func enableAutostart() error { return errNoAutostart }

func disableAutostart() error { return errNoAutostart }
