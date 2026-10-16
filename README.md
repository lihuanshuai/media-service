# MediaServiceTray

系统托盘小工具：单击托盘图标即可一键「启动 / 停止」Windows 的
**Windows Media Player Network Sharing Service**（`WMPNetworkSvc`）服务，
用于快速切换网络共享 / 媒体流功能，不需要每次打开服务管理器。

## 功能

- **左键单击托盘图标**：服务运行中 → 停止；已停止 → 启动
- **右键菜单**：
  - 当前服务状态
  - 启动服务 / 停止服务
  - 开机自启（勾选）
  - 退出
- **图标颜色**表示状态：绿色 = 运行中，灰色 = 已停止，琥珀色 = 切换中 / 未知
- 每 3 秒自动刷新状态，服务被外部改动也会实时反映
- **单实例**：重复启动不会出现多个托盘图标
- **开机自启**：登录时静默以最高权限启动，不弹 UAC 确认框

## 权限说明

启动 / 停止服务需要管理员权限。程序启动时会自动以管理员身份重新拉起
（首次会弹一次 UAC 确认），之后在托盘内操作不会再弹。

## 构建

需要 Go 1.21+（Windows）。

```powershell
powershell -ExecutionPolicy Bypass -File build.ps1
```

产物位于 `bin\MediaServiceTray.exe`（GUI 子系统，无黑色控制台窗口）。

### 手动构建

```powershell
go build -trimpath -ldflags "-s -w -H windowsgui" -o bin\MediaServiceTray.exe .
```

### 开发调试

跳过自动提权（此时启动 / 停止会失败，仅供本地联调界面）：

```powershell
go run . -no-elevate
```

## 图标

托盘图标由 `tools/genicons` 生成（纯标准库，无外部依赖），
三态图标均嵌入到可执行文件中：

```powershell
go run ./tools/genicons
```

## 开机自启机制

勾选「开机自启」会创建名为 `MediaServiceTray` 的计划任务：

- 触发器：当前用户登录时（`LogonTrigger`）
- 运行级别：最高权限（`HighestAvailable`）
- 登录类型：交互式令牌（`InteractiveToken`）

这样登录时托盘会静默以管理员权限启动，无需 UAC 确认。

如需手动删除自启任务：

```powershell
schtasks /Delete /TN "\MediaServiceTray" /F
```

## 测试

```powershell
go test ./...
```
