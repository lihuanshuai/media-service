//go:build !windows

package main

import "errors"

var errUnsupportedOS = errors.New("mediaservice 仅支持 Windows")

func queryService() (svcState, error) { return stateUnknown, errUnsupportedOS }

func startService() error { return errUnsupportedOS }

func stopService() error { return errUnsupportedOS }
