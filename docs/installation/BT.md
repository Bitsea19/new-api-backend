# 宝塔面板部署教程

本文档提供使用宝塔面板 Docker 功能部署 New API 的图文教程。

> 📖 官方文档：[宝塔面板部署](https://docs.newapi.pro/zh/docs/installation/deployment-methods/bt-docker-installation)

> **本仓库是前后端分离的二开版本**，后端本身不含界面。部署完成后，还需要单独部署前端面板（见 [new-api-web 仓库](https://github.com/Bitsea19/new-api-web)）。

***

## 前置要求

| 项目    | 要求                                 |
| ----- | ---------------------------------- |
| 宝塔面板  | ≥ 9.2.0 版本                         |
| 推荐系统  | CentOS 7+、Ubuntu 18.04+、Debian 10+ |
| 服务器配置 | 至少 1 核 2G 内存                       |

***

## 步骤一：安装宝塔面板

1. 前往 [宝塔面板官网](https://www.bt.cn/new/download.html) 下载适合您系统的安装脚本
2. 运行安装脚本安装宝塔面板
3. 安装完成后，使用提供的地址、用户名和密码登录宝塔面板

***

## 步骤二：安装 Docker

1. 登录宝塔面板后，在左侧菜单栏找到并点击 **Docker**
2. 首次进入会提示安装 Docker 服务，点击 **立即安装**
3. 按照提示完成 Docker 服务的安装

***

## 步骤三：安装 New API

> **推荐做法：从本仓库源码本地构建镜像。** 官方镜像 `calciumion/new-api:latest` 是上游版本，**不包含**本二开版本的前后端分离改动（`FRONTEND_BASE_URL`、纯 API 二进制、自定义 CORS 等），直接拉取会得到不一致的行为。

### 方法一：从源码构建并用 Compose 启动（推荐）

1. 在宝塔面板中创建网站目录，如 `/www/wwwroot/new-api-backend`
2. 上传本仓库源码到该目录（可用 `scripts/package-for-server.ps1` 或 `scripts/package-for-server.sh` 打包后上传）
3. 编辑目录下的 `docker-compose.yml`：

```yaml
services:
  new-api:
    build:
      context: .
      dockerfile: Dockerfile
    image: new-api:local
    container_name: new-api
    restart: always
    ports:
      - "127.0.0.1:3000:3000"
    volumes:
      - ./data:/data
    environment:
      - SESSION_SECRET=your_session_secret_here  # 请修改为随机字符串
      - TZ=Asia/Shanghai
      - FRONTEND_BASE_URL=https://app.example.com     # 前端面板的公开地址
      - API_BASE_URL=https://api.example.com          # 后端自身的公开地址
      - CORS_ALLOWED_ORIGINS=https://app.example.com  # 允许跨域携带 Cookie 的精确 origin
```

1. 在终端中进入目录并启动：

```bash
cd /www/wwwroot/new-api-backend
docker compose up -d --build
docker compose logs -f new-api
```

1. 验证后端：

```bash
curl http://127.0.0.1:3000/api/status
```

### 方法二：使用宝塔应用商店

> ⚠️ 应用商店安装的是**上游官方镜像**，不含本二开版本的分离部署改动。如需本仓库的功能，请使用方法一。

1. 在宝塔面板 Docker 功能中，点击 **应用商店**
2. 搜索并找到 **New-API**
3. 点击 **安装**
4. 配置以下基本选项：
   - **容器名称**：可自定义，默认为 `new-api`
   - **端口映射**：默认为 `3000:3000`
   - **环境变量**：
     - `SESSION_SECRET`：会话密钥（**必填**，多机部署时必须一致）
     - `CRYPTO_SECRET`：加密密钥（使用 Redis 时必填）
5. 点击 **确认** 开始安装
6. 等待安装完成后，访问 `http://您的服务器IP:3000/api/status` 即可确认后端已就绪

***

## 配置说明

### 必要环境变量

| 变量名                 | 说明                 | 是否必填   |
| ------------------- | ------------------ | ------ |
| `SESSION_SECRET`    | 会话密钥，多机部署必须一致      | **必填** |
| `CRYPTO_SECRET`     | 加密密钥，使用 Redis 时必填  | 条件必填   |
| `SQL_DSN`           | 数据库连接字符串（使用外部数据库时） | 可选     |
| `REDIS_CONN_STRING` | Redis 连接字符串        | 可选     |

### 前后端分离相关环境变量

前后端部署在不同地址时必须配置，详见[服务器部署指南](../deployment/server-backend.md)：

| 变量名                   | 说明                             | 是否必填   |
| --------------------- | ------------------------------ | ------ |
| `FRONTEND_BASE_URL`   | 前端面板的公开地址，浏览器导航重定向与各类回跳都指向它 | 分离部署时必填 |
| `API_BASE_URL`        | 后端自身的公开地址                    | 分离部署时必填 |
| `CORS_ALLOWED_ORIGINS` | 允许携带 Cookie 跨域访问 API 的精确 origin   | 前后端不同源时必填 |
| `SESSION_COOKIE_SECURE` | HTTPS 环境下设为 `true`              | 生产必填   |

### 生成随机密钥

```bash
# 生成 SESSION_SECRET
openssl rand -hex 16

# 或使用 Linux 命令
head -c 16 /dev/urandom | xxd -p
```

***

## 常见问题

### Q1：无法访问 3000 端口？

1. 检查服务器防火墙是否开放 3000 端口
2. 在宝塔面板 **安全** 中放行 3000 端口
3. 检查云服务器安全组是否开放端口

### Q2：登录后提示会话失效？

确保设置了 `SESSION_SECRET` 环境变量，且值不为空。

### Q3：数据如何持久化？

使用 Docker 卷映射数据目录：

```yaml
volumes:
  - ./data:/data
```

### Q4：如何更新版本？

本仓库从源码构建，更新步骤是拉取新代码后重新构建：

```bash
cd /www/wwwroot/new-api-backend
git pull                       # 或重新上传源码压缩包
docker compose up -d --build
```

> 使用官方镜像的部署请改用 `docker pull calciumion/new-api:latest` + `docker-compose down && docker-compose up -d`。

***

## 相关链接

- [服务器部署指南](../deployment/server-backend.md)
- [前端部署指南](https://github.com/Bitsea19/new-api-web/blob/main/deploy/DEPLOYMENT.md)
- [官方文档](https://docs.newapi.pro/zh/docs/installation)
- [环境变量配置](https://docs.newapi.pro/zh/docs/installation/config-maintenance/environment-variables)
- [常见问题](https://docs.newapi.pro/zh/docs/support/faq)
- [GitHub 仓库](https://github.com/Bitsea19/new-api-backend)

***

## 截图示例

![宝塔面板 Docker 安装](https://github.com/user-attachments/assets/7a6fc03e-c457-45e4-b8f9-184508fc26b0)

> ⚠️ 注意：密钥为环境变量 `SESSION_SECRET`，请务必设置！

