package main

import (
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/systray"
)

// app 负责托盘图标、右键菜单与服务状态之间的同步。
type app struct {
	mu       sync.Mutex
	state    svcState
	busyText string

	busy atomic.Bool // 是否有启动/停止操作正在执行

	rendered string // 上一次渲染的状态签名，用于跳过重复刷新

	statusItem    *systray.MenuItem
	startItem     *systray.MenuItem
	stopItem      *systray.MenuItem
	autostartItem *systray.MenuItem
	quitItem      *systray.MenuItem
}

var appInstance *app

const pollInterval = 3 * time.Second

// onReady 在托盘就绪后构建菜单并启动后台轮询。
func onReady() {
	systray.SetIcon(iconUnknown)
	systray.SetTooltip("WMPNetworkSvc：正在检测状态…")
	systray.SetOnTapped(onLeftTapped)

	appInstance = &app{}
	appInstance.buildMenu()
	appInstance.refresh()

	go appInstance.menuLoop()
	go appInstance.pollLoop()
}

func onLeftTapped() {
	if appInstance == nil {
		return
	}
	go appInstance.toggle()
}

// toggle 根据当前状态切换服务：运行则停止，已停止则启动。
func (a *app) toggle() {
	a.mu.Lock()
	st := a.state
	a.mu.Unlock()

	switch st {
	case stateRunning:
		a.runOp("停止", stopService)
	case stateStopped:
		a.runOp("启动", startService)
	default:
		a.refresh()
	}
}

// runOp 串行执行启动/停止操作，并在期间更新界面状态。
func (a *app) runOp(label string, op func() error) {
	if !a.busy.CompareAndSwap(false, true) {
		return
	}
	defer func() {
		a.busy.Store(false)
		a.refresh()
	}()

	a.setBusyText(label + "中…")
	a.render()
	if err := op(); err != nil {
		msgBox(label+"失败", err.Error())
	}
}

func (a *app) setBusyText(text string) {
	a.mu.Lock()
	a.busyText = text
	a.mu.Unlock()
}

// refresh 查询服务最新状态并重绘托盘。
func (a *app) refresh() {
	st, err := queryService()
	a.mu.Lock()
	if err != nil {
		a.state = stateUnknown
	} else {
		a.state = st
	}
	a.mu.Unlock()
	a.render()
}

// render 根据当前状态更新图标、提示文字与菜单可用性。
// 状态未发生变化时直接返回，避免定时轮询反复刷新导致托盘图标闪烁。
func (a *app) render() {
	busy := a.busy.Load()

	a.mu.Lock()
	st := a.state
	busyText := a.busyText

	signature := st.String()
	if busy {
		signature = "busy:" + busyText
	}
	if a.rendered == signature {
		a.mu.Unlock()
		return
	}
	a.rendered = signature
	a.mu.Unlock()

	label := "WMPNetworkSvc：" + st.String()
	if busy {
		label = "WMPNetworkSvc：" + busyText
	}

	switch {
	case busy:
		systray.SetIcon(iconUnknown)
		systray.SetTooltip(label + "\n请稍候…")
	case st == stateRunning:
		systray.SetIcon(iconOn)
		systray.SetTooltip(label + "\n左键单击停止，右键查看更多操作")
	case st == stateStopped:
		systray.SetIcon(iconOff)
		systray.SetTooltip(label + "\n左键单击启动，右键查看更多操作")
	default:
		systray.SetIcon(iconUnknown)
		systray.SetTooltip(label)
	}

	a.statusItem.SetTitle(label)
	if busy || (st != stateStopped && st != stateUnknown) {
		a.startItem.Disable()
	} else {
		a.startItem.Enable()
	}
	if busy || st != stateRunning {
		a.stopItem.Disable()
	} else {
		a.stopItem.Enable()
	}
}

func (a *app) menuLoop() {
	for {
		select {
		case <-a.startItem.ClickedCh:
			a.runOp("启动", startService)
		case <-a.stopItem.ClickedCh:
			a.runOp("停止", stopService)
		case <-a.autostartItem.ClickedCh:
			a.toggleAutostart()
		case <-a.quitItem.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func (a *app) pollLoop() {
	t := time.NewTicker(pollInterval)
	defer t.Stop()
	for range t.C {
		if !a.busy.Load() {
			a.refresh()
		}
	}
}

func (a *app) toggleAutostart() {
	enabled := autostartEnabled()
	var err error
	if enabled {
		err = disableAutostart()
	} else {
		err = enableAutostart()
	}
	if err != nil {
		msgBox("开机自启", err.Error())
	}
	if autostartEnabled() {
		a.autostartItem.Check()
	} else {
		a.autostartItem.Uncheck()
	}
}

func (a *app) buildMenu() {
	a.statusItem = systray.AddMenuItem("WMPNetworkSvc：…", "当前服务状态")
	a.statusItem.Disable()
	systray.AddSeparator()
	a.startItem = systray.AddMenuItem("启动服务", "启动 WMPNetworkSvc")
	a.stopItem = systray.AddMenuItem("停止服务", "停止 WMPNetworkSvc")
	systray.AddSeparator()
	a.autostartItem = systray.AddMenuItemCheckbox("开机自启", "登录时自动运行本工具", autostartEnabled())
	systray.AddSeparator()
	a.quitItem = systray.AddMenuItem("退出", "退出本工具")
}
