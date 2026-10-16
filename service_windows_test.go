//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows/svc"
)

func TestFromWinState(t *testing.T) {
	cases := []struct {
		in   svc.State
		want svcState
	}{
		{svc.Running, stateRunning},
		{svc.Stopped, stateStopped},
		{svc.StartPending, statePending},
		{svc.StopPending, statePending},
		{svc.ContinuePending, statePending},
		{svc.PausePending, statePending},
		{svc.Paused, statePaused},
		{svc.State(99), stateUnknown},
	}
	for _, c := range cases {
		if got := fromWinState(c.in); got != c.want {
			t.Errorf("fromWinState(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
