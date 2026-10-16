//go:build windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ensureElevated 确保程序以管理员权限运行：
//   - 已提权时返回 true，由当前进程继续运行；
//   - 未提权时通过 ShellExecute("runas") 以管理员身份重新拉起自身并返回 false，
//     由新实例接管；用户取消 UAC 或拉起失败时会弹出提示。
func ensureElevated() bool {
	if windows.GetCurrentProcessToken().IsElevated() {
		return true
	}
	// 已有实例在运行时直接退出，避免为了提示“已在运行”再弹一次 UAC。
	if singleInstanceRunning() {
		msgBox("已在运行", "MediaServiceTray 已经在系统托盘中运行了。")
		return false
	}
	if err := relaunchElevated(); err != nil {
		reason := "以管理员身份重新启动失败：" + err.Error()
		if errors.Is(err, windows.ERROR_CANCELLED) {
			reason = "已取消管理员授权。"
		}
		msgBox("需要管理员权限", "启动/停止 WMPNetworkSvc 需要管理员权限。\n\n"+reason)
		return false
	}
	return false
}

func relaunchElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	cwd, err := windows.UTF16PtrFromString(filepath.Dir(exe))
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, nil, cwd, windows.SW_SHOWNORMAL)
}

var (
	user32          = windows.NewLazySystemDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

// msgBox 弹出模态消息框，用于向用户提示错误等信息。
func msgBox(title, text string) {
	const (
		mbIconError     = 0x00000010
		mbSetForeground = 0x00010000
	)

	t, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	m, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), mbIconError|mbSetForeground)
}
