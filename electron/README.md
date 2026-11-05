# New API 桌面端（Electron）

本目录包含 New API 的 Electron 封装，为 Windows、macOS 和 Linux 提供带系统托盘支持的原生桌面应用。

> **注意**：前端源码位于独立仓库 `new-api-web`。本目录的构建脚本会从 `../new-api-web` 拉取前端产物（可用 `FRONTEND_DIR` 覆盖）。

## 前置条件

### 1. Go 二进制（必需）

Electron 应用需要已编译的 Go 二进制才能工作。有两种方式：

**方式 A：使用已有二进制（无需安装 Go）**

```bash
# 如果你手头已有预编译的二进制（例如 new-api-macos）
cp ../new-api-macos ../new-api
```

**方式 B：从源码构建（需要 Go）**

```bash
# 在仓库根目录
go build -o new-api .
```

### 2. Electron 依赖

```bash
cd electron
npm install
```

## 开发

分别在三个终端中启动后端、前端与 Electron：

```bash
# 仓库根目录：后端
go run main.go
```

```bash
# 仓库根目录：前端
make dev-web
```

```bash
# electron/ 目录
npm run dev-app
```

这会：

- 使用 3000 端口上的 Go 后端
- 使用 5173 端口上的 Rsbuild 前端开发服务器
- 打开一个启用了开发者工具的 Electron 窗口
- 创建系统托盘图标（macOS 上位于菜单栏）
- 把数据库存放在 `../data/new-api.db`

## 生产构建

### 快速构建

```bash
# 在 electron/ 目录下，构建前端、Go 二进制与桌面安装包
./build.sh
```

```bash
# 或为当前平台打包已有二进制
npm run build
```

```bash
# 各平台构建
npm run build:mac    # 生成 .dmg 与 .zip
npm run build:win    # 生成 .exe 安装包
npm run build:linux  # 生成 .AppImage 与 .deb
```

### 构建产物

- 构建出的应用位于 `electron/dist/`
- macOS：`.dmg`（安装包）与 `.zip`（便携版）
- Windows：`.exe`（安装包）与便携版 exe
- Linux：`.AppImage` 与 `.deb`

## 配置

### 端口

默认端口为 3000。要修改，编辑 `main.js`：

```javascript
const PORT = 3000; // 改成期望的端口
```

### 数据库位置

- **开发环境**：`../data/new-api.db`（项目目录内）
- **生产环境**：
  - macOS：`~/Library/Application Support/New API/data/`
  - Windows：`%APPDATA%/New API/data/`
  - Linux：`~/.config/New API/data/`
