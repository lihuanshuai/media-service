//go:build !windows

package main

func run(noElevate bool) error {
	return errUnsupportedOS
}
