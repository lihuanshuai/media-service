//go:build windows

package main

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// serviceName 是 Windows Media Player 网络共享服务。
const serviceName = "WMPNetworkSvc"

const (
	startTimeout = 30 * time.Second
	stopTimeout  = 30 * time.Second
)

// queryService 查询 WMPNetworkSvc 的当前状态。
func queryService() (svcState, error) {
	m, err := mgr.Connect()
	if err != nil {
		return stateUnknown, fmt.Errorf("连接服务管理器失败：%w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return stateUnknown, fmt.Errorf("打开服务 %s 失败：%w", serviceName, err)
	}
	defer s.Close()

	st, err := s.Query()
	if err != nil {
		return stateUnknown, fmt.Errorf("查询服务状态失败：%w", err)
	}
	return fromWinState(st.State), nil
}

// startService 启动服务并等待其进入运行状态。
func startService() error {
	m, s, err := openService()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	defer s.Close()

	st, err := s.Query()
	if err != nil {
		return fmt.Errorf("查询服务状态失败：%w", err)
	}
	if st.State == svc.Running {
		return nil
	}

	if err := s.Start(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
		return fmt.Errorf("启动服务失败：%w", err)
	}
	return waitForState(s, svc.Running, startTimeout)
}

// stopService 停止服务并等待其完全停止。
func stopService() error {
	m, s, err := openService()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	defer s.Close()

	st, err := s.Query()
	if err != nil {
		return fmt.Errorf("查询服务状态失败：%w", err)
	}
	if st.State == svc.Stopped {
		return nil
	}

	// 服务尚在启动过程时 Control 会返回 ERROR_SERVICE_CANNOT_ACCEPT_CTRL，稍后重试。
	var ctrlErr error
	for attempt := 0; attempt < 10; attempt++ {
		if _, ctrlErr = s.Control(svc.Stop); ctrlErr == nil {
			break
		}
		if !errors.Is(ctrlErr, windows.ERROR_SERVICE_CANNOT_ACCEPT_CTRL) {
			return fmt.Errorf("发送停止指令失败：%w", ctrlErr)
		}
		time.Sleep(500 * time.Millisecond)
	}
	if ctrlErr != nil {
		return fmt.Errorf("发送停止指令失败：%w", ctrlErr)
	}
	return waitForState(s, svc.Stopped, stopTimeout)
}

func openService() (*mgr.Mgr, *mgr.Service, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, nil, fmt.Errorf("连接服务管理器失败：%w", err)
	}
	s, err := m.OpenService(serviceName)
	if err != nil {
		m.Disconnect()
		return nil, nil, fmt.Errorf("打开服务 %s 失败：%w", serviceName, err)
	}
	return m, s, nil
}

// waitForState 轮询服务状态，直到达到期望状态或超时。
func waitForState(s *mgr.Service, want svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		st, err := s.Query()
		if err != nil {
			return fmt.Errorf("查询服务状态失败：%w", err)
		}
		if st.State == want {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("等待服务进入 %v 超时（当前状态 %v）", want, st.State)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// fromWinState 把 Windows 服务状态映射为界面使用的简化状态。
func fromWinState(s svc.State) svcState {
	switch s {
	case svc.Running:
		return stateRunning
	case svc.Stopped:
		return stateStopped
	case svc.StartPending, svc.StopPending, svc.ContinuePending, svc.PausePending:
		return statePending
	case svc.Paused:
		return statePaused
	default:
		return stateUnknown
	}
}
