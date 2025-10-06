# BTC实时价格监控 🚀

一个使用Wails（Go + Vue3）开发的比特币实时价格监控应用，支持系统托盘显示。

## ✨ 功能特性

- 🔄 **实时价格更新**：每10秒自动从币安API获取最新BTC价格
- 📊 **现代化UI**：使用Vue3构建的漂亮界面，支持响应式设计
- 🔔 **托盘显示**：窗口标题实时显示BTC价格，方便监控
- ⚡ **轻量高效**：使用Go语言后端，性能优异
- 🎨 **毛玻璃效果**：现代化的UI设计，支持渐变背景

## 🛠️ 技术栈

**后端**
- Go 1.23
- Wails v2.10.2
- 币安公开API

**前端**
- Vue 3 + Vite
- 现代CSS（Flexbox、动画、毛玻璃效果）

## 🚀 快速开始

### 直接运行（推荐）
项目根目录下的 `coin_price_window.exe` 是已经打包好的可执行文件，**可以直接运行**！

> 💡 **注意**：国内用户需要先配置代理才能访问币安API，请参考下方的"配置"章节。

### 开发模式
```bash
wails dev
```

### 构建生产版本
```bash
wails build
```

## 📖 使用说明

### 功能说明
- 应用启动后自动获取BTC价格
- 窗口标题显示实时价格（格式：BTC: $价格）
- 主界面显示详细价格信息和更新时间
- 每10秒自动更新一次

### 窗口操作
- 点击关闭按钮会隐藏窗口但不退出程序
- 从任务栏可以恢复窗口

## 📁 核心文件

- `main.go` - 主入口和窗口配置
- `app.go` - 价格获取逻辑
- `tray.go` - 托盘管理器
- `frontend/src/App.vue` - 前端UI

## ⚙️ 配置

### 代理配置（重要！）

**国内用户必须配置代理才能访问币安API！**

应用支持3种代理配置方式（按优先级）：

#### 方法1：配置文件（推荐）
在程序目录创建或编辑 `config.json`：
```json
{
  "proxy": {
    "enabled": true,
    "address": "http://127.0.0.1:7897"
  },
  "update_interval": 10
}
```

常见代理端口：
- Clash for Windows: `http://127.0.0.1:7897` (默认)
- Clash Verge: `http://127.0.0.1:7890`
- V2rayN: `http://127.0.0.1:10809`

#### 方法2：环境变量
设置系统环境变量：
```bash
# Windows PowerShell
$env:HTTP_PROXY="http://127.0.0.1:7897"
$env:HTTPS_PROXY="http://127.0.0.1:7897"

# Linux/Mac
export HTTP_PROXY=http://127.0.0.1:7897
export HTTPS_PROXY=http://127.0.0.1:7897
```

#### 方法3：Clash TUN模式
如果以上方法都不行，开启Clash的TUN模式即可全局代理。

### 配置说明

- **proxy.enabled**: 是否启用代理（true/false）
- **proxy.address**: 代理地址（如 `http://127.0.0.1:7897`）
- **update_interval**: 价格更新间隔（秒），建议10-60秒

### 验证代理是否生效

启动程序后，查看控制台输出：
- ✓ 使用配置代理: http://127.0.0.1:7897 - 成功
- ✗ 代理地址解析失败 - 代理地址格式错误
- ⚠ 未配置代理 - 国内无法访问

## 📝 项目结构
```
coin_price_window/
├── main.go              # 主入口
├── app.go               # 业务逻辑
├── tray.go              # 托盘管理
├── frontend/
│   └── src/
│       └── App.vue      # Vue界面
└── build/               # 构建输出
```

---

**Powered by Wails + Vue3**
