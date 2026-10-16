//go:build windows

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"time"
)

// 开机自启通过计划任务实现：登录时以最高权限启动本程序。
// 相比注册表 Run 键，计划任务可以在登录时静默提权，避免每次都弹 UAC 确认框。
const autostartTaskName = `\MediaServiceTray`

// autostartEnabled 判断开机自启的计划任务是否存在。
func autostartEnabled() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "schtasks.exe", "/Query", "/TN", autostartTaskName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}

// enableAutostart 创建（或覆盖）开机自启的计划任务。
func enableAutostart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("获取当前用户信息失败：%w", err)
	}

	tmp, err := os.CreateTemp("", "mediaservice-autostart-*.xml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if err := writeUTF16LE(tmp, buildAutostartTaskXML(exe, u.Uid)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return runSchTasks("/Create", "/TN", autostartTaskName, "/XML", tmp.Name(), "/F")
}

// disableAutostart 删除开机自启的计划任务。
func disableAutostart() error {
	return runSchTasks("/Delete", "/TN", autostartTaskName, "/F")
}

func runSchTasks(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "schtasks.exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("执行 schtasks 失败：%w（%s）", err, msg)
		}
		return fmt.Errorf("执行 schtasks 失败：%w", err)
	}
	return nil
}

// buildAutostartTaskXML 生成「登录时以最高权限运行」的计划任务 XML。
func buildAutostartTaskXML(exe, sid string) string {
	const tmpl = `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>WMPNetworkSvc 系统托盘工具（登录时自动启动）</Description>
    <URI>%TASK%</URI>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      <UserId>%SID%</UserId>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>%SID%</UserId>
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>false</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <IdleSettings>
      <StopOnIdleEnd>false</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Priority>7</Priority>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>%CMD%</Command>
      <Arguments></Arguments>
    </Exec>
  </Actions>
</Task>
`
	return strings.NewReplacer(
		"%TASK%", autostartTaskName,
		"%SID%", sid,
		"%CMD%", xmlEscape(exe),
	).Replace(tmpl)
}

func xmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	).Replace(s)
}

// writeUTF16LE 以 UTF-16 LE（含 BOM）写出文本，schtasks 导入 XML 时要求该编码。
func writeUTF16LE(f *os.File, s string) error {
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xFE})
	for _, r := range s {
		if r <= 0xFFFF {
			buf.WriteByte(byte(r))
			buf.WriteByte(byte(r >> 8))
			continue
		}
		r -= 0x10000
		hi := 0xD800 + (r >> 10)
		lo := 0xDC00 + (r & 0x3FF)
		buf.WriteByte(byte(hi))
		buf.WriteByte(byte(hi >> 8))
		buf.WriteByte(byte(lo))
		buf.WriteByte(byte(lo >> 8))
	}
	_, err := f.Write(buf.Bytes())
	return err
}
