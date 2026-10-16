package main

// svcState 是服务状态的简化视图，供托盘界面使用。
type svcState int

const (
	stateUnknown svcState = iota // 查询失败或状态无法确定
	stateRunning                 // 运行中
	stateStopped                 // 已停止
	statePending                 // 启动/停止切换中
	statePaused                  // 已暂停
)

func (s svcState) String() string {
	switch s {
	case stateRunning:
		return "运行中"
	case stateStopped:
		return "已停止"
	case statePending:
		return "切换中"
	case statePaused:
		return "已暂停"
	default:
		return "未知"
	}
}
