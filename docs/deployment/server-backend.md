# 后端服务器部署指南（源码本地构建）

后端是一个**纯 API 服务**，不带前端面板。面板（dashboard）单独部署，见 `new-api-web` 仓库。

本文说明如何把后端源码上传到服务器，并在服务器上**本地构建镜像**，不使用远程的 `calciumion/new-api:latest`。

---

## 1. 架构

```
浏览器
  │
  ├─ app.example.com  ──▶  前端容器 (new-api-web, Nginx + 静态资源)
  │                          └─ JS 里的 VITE_API_BASE_URL 指向 api.example.com
  │
  └─ api.example.com  ──▶  后端容器 (new-api, Go API 二进制)
                             ├─▶ postgres
                             └─▶ redis
```

后端根路径 `/` 会 301 跳转到 `FRONTEND_BASE_URL`。

---

## 2. 打包源码（本地）

```bash
# Windows PowerShell
powershell -ExecutionPolicy Bypass -File scripts\package-for-server.ps1

# Linux / macOS
./scripts/package-for-server.sh
```

产物在仓库的**上一级目录**：`new-api-backend-src.tar.gz`

脚本打包的是 **git 已跟踪文件的当前内容**，所以本地未提交的改动也会被打进去。
被排除的只有前端代码 —— 前端在独立的 `new-api-web` 仓库，不属于后端。

> ⚠️ 如果你用 `zip` 而不是这个脚本，记得手动确认包里**没有** `web/` 目录。

---

## 3. 上传并解压

```bash
scp new-api-backend-src.tar.gz root@<服务器IP>:/opt/
```

服务器上：

```bash
cd /opt
tar -xzf new-api-backend-src.tar.gz
cd /opt/new-api-backend
```

---

## 4. 配置环境变量

`docker-compose.yml` 里大部分环境变量是注释掉的，上线前必须打开：

```yaml
environment:
  - SQL_DSN=postgresql://root:123456@postgres:5432/new-api  # 改密码
  - REDIS_CONN_STRING=redis://:123456@redis:6379               # 改密码
  - SESSION_SECRET=<随机字符串>                                  # 必填
  - FRONTEND_BASE_URL=https://app.example.com
  - API_BASE_URL=https://api.example.com
  - CORS_ALLOWED_ORIGINS=https://app.example.com
  - SESSION_COOKIE_SECURE=true
  - SESSION_COOKIE_TRUSTED_URL=https://app.example.com
```

生成随机 `SESSION_SECRET`：

```bash
openssl rand -base64 32
```

> **关于 `SESSION_COOKIE_SECURE`**：设为 `true` 时，后端会启用 Secure Refresh Cookie 和严格的
> refresh/logout Origin 校验。此时 `SESSION_COOKIE_TRUSTED_URL` 是**必填**的精确 HTTPS Origin。
> 如果暂时只有 HTTP 域名，先留空（保持注释），功能正常但会打印安全告警。

---

## 5. 构建并启动

```bash
cd /opt/new-api-backend
docker compose up -d --build
```

`--build` 会用本目录的 `Dockerfile` 编译 Go 二进制，产出镜像 `new-api:local`（**本地 tag，不推送**）。

关于镜像来源的说明：

- `docker-compose.yml` 里配的是 `build: { context: . }`，所以 `docker compose up -d` 也会走本地构建，
  **不会**去 Docker Hub 拉 `calciumion/new-api:latest`。
- 如果之前已经拉过远程镜像，容器里跑的还是远程版本，先停掉重来：

```bash
docker compose down
docker compose up -d --build
```

---

## 6. 验证

```bash
# 镜像必须是本地构建的
docker images | grep new-api
# new-api   local   ...   <构建时间>

# 容器运行的是本地镜像
docker inspect new-api --format '{{.Config.Image}}'
# new-api:local

# API 正常
curl -s http://127.0.0.1:3000/api/status
# {"success":true,...}
```

> **数据库首次初始化**的日志里，`connection refused` 是正常的 —— 那是 API 比数据库先启动，
> 重启后就正常了。compose 里已经加了 `condition: service_healthy`，首次初始化后不会再出现。

> **版本号**：源码包里的 `VERSION` 文件是空的（正常，它由 CI 写入）。
> 镜像里的版本号会是 `dev`（或你用 `docker compose build --build-arg APP_VERSION=v1.0.0` 传入的值）。

> **日志目录**：容器以 root 运行，root 可以写 `/app/logs`，但宿主机上 `./logs` 会生成 root 属主的文件。

---

## 7. 前端部署（另一台服务器或同一台）

见 `new-api-web` 仓库的 `deploy/DEPLOYMENT.md`。

要点：

- 前端镜像构建时通过 build-arg 传入 API 地址：
  `VITE_API_BASE_URL=https://api.example.com`
- 前端 Nginx 反向代理后端 API 时**必须**关闭 `proxy_buffering`，否则 SSE 流式响应会被缓冲。

---

## 8. 常见问题

### 容器日志里版本是 `v1.0.0-rc.48`

说明跑的还是 Docker Hub 的远程镜像。检查：

```bash
docker inspect new-api --format '{{.Config.Image}}'
```

如果输出 `calciumion/new-api:latest`，说明 `docker-compose.yml` 没同步到服务器。

### 构建报错 `failed to solve: process ... did not complete successfully`

Go 编译需要下载依赖，服务器需要能访问 `proxy.golang.org`。国内服务器加代理：

```bash
export GOPROXY=https://goproxy.cn,direct
```

或者给 Docker 加代理配置。

### `docker compose up -d --build` 报 `version is obsolete`

`docker compose` (v2) 不再需要 `version` 字段，旧写法会打印告警但不影响功能。

### 日志里有 `TRUSTED_PROXIES is unset or blank`

未配置时默认信任回环/RFC1918 地址。对外暴露时应显式配置：

```bash
- TRUSTED_PROXIES=172.20.0.0/16
```

### 容器一直重启

```bash
docker compose logs new-api | tail -50
```

常见原因：`SQL_DSN` 密码不一致、数据库还在首次初始化、`SESSION_SECRET` 没设置。

---

## 9. 加速构建

Go 模块下载是构建最慢的部分。服务器上如果经常需要重新构建，可以先预热：

```bash
# 预热 Go 模块缓存（首次会下载全部依赖，比较慢）
docker compose build new-api

# 之后正常的增量构建会快很多
docker compose up -d --build
```

更快的做法是给 Docker BuildKit 配上 registry 缓存：

```bash
docker buildx build --cache-from=type=registry,ref=your-registry/new-api:cache .
```

> 📌 用好这个指南，方便新老用户快速部署。参考 Claude/ChatGPT 官方文档的写法风格。
