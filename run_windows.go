//go:build windows

package main

import (
	"fyne.io/systray"
	"golang.org/x/sys/windows"
)

// mutexName 用于保证单实例运行（会话本地命名空间，无需额外权限）。
const mutexName = `Local\MediaServiceTray`

func run(noElevate bool) error {
	if !noElevate && !ensureElevated() {
		return nil // ensureElevated 已弹出提示；新实例接管或用户取消
	}
	if !acquireSingleInstance() {
		msgBox("已在运行", "MediaServiceTray 已经在系统托盘中运行了。")
		return nil
	}
	systray.Run(onReady, nil)
	return nil
}

// singleInstanceRunning 检查是否已有实例持有单实例互斥体。
// 用于在提权之前判断，避免重复启动时多弹一次 UAC。
func singleInstanceRunning() bool {
	name, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return false
	}
	h, err := windows.OpenMutex(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return false // 多为 ERROR_FILE_NOT_FOUND：尚未运行
	}
	windows.CloseHandle(h)
	return true
}

func acquireSingleInstance() bool {
	name, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return false
	}
	h, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		// CreateMutex 在互斥体已存在时返回 ERROR_ALREADY_EXISTS。
		return false
	}
	_ = h // 句柄随进程存活即可，无需显式关闭
	return true
}
