<div align="center">

![new-api](./assets/logo.png)

# New API · 后端

**连接模型、应用与 Agent 的 AI 网关 —— 纯 API 服务**

[![license](https://img.shields.io/github/license/QuantumNous/new-api?color=brightgreen)](https://github.com/QuantumNous/new-api/blob/main/LICENSE)
[![go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)](./go.mod)

[前端面板 new-api-web](https://github.com/Bitsea19/new-api-web) · [官方文档](https://docs.newapi.ai/zh/docs) · [问题反馈](https://github.com/QuantumNous/new-api/issues)

</div>

---

## 目录

- [这是什么](#这是什么)
- [与上游的差异](#与上游的差异)
- [快速开始](#快速开始)
- [配置](#配置)
- [部署](#部署)
- [核心能力](#核心能力)
- [开发](#开发)
- [同步上游](#同步上游)
- [文档索引](#文档索引)
- [合规声明](#合规声明)
- [许可证](#许可证)

---

## 这是什么

New API 是面向应用、Agent 和团队的自托管 AI 网关。将不同厂商的模型服务接入统一入口，在同一套控制台中管理渠道、访问权限、用量与成本。

支持接入 OpenAI、Anthropic、Google Gemini、Azure OpenAI、AWS Bedrock、Vertex AI、DeepSeek、通义千问及其他兼容服务。

**本仓库只包含后端**，是一个不携带任何前端资源的纯 API 服务：

```
┌─────────────────────────┐         ┌─────────────────────────┐
│  new-api-web（前端）      │  ──▶   │  new-api-backend（本仓库） │
│  React SPA + Nginx       │  HTTP  │  Go API 服务              │
│  独立构建、独立部署        │◀──     │  独立构建、独立部署        │
└─────────────────────────┘         └───────────┬─────────────┘
                                                 │
                                    ┌────────────┴────────────┐
                                    │                         │
                              ┌─────▼─────┐           ┌───────▼──────┐
                              │ PostgreSQL │           │    Redis     │
                              └───────────┘           └──────────────┘
```

> 💡 本仓库是 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 的二次开发版本。
> 主要改动是**将前端面板完全剥离**，面板已迁移到独立仓库 [new-api-web](https://github.com/Bitsea19/new-api-web)，
> 两个仓库可以分别构建、分别部署、分别升级。

---

## 与上游的差异

### 前后端彻底分离

| 维度 | 上游原版 | 本二开版本 |
| --- | --- | --- |
| 前端代码位置 | `web/` 目录，与后端同仓库 | **独立仓库** [`new-api-web`](https://github.com/Bitsea19/new-api-web) |
| 后端构建产物 | 需要前端产物才能构建 | `go build` 直接产出纯 API 二进制 |
| 默认行为 | 托管内置前端面板 | **只提供 API**，不托管任何静态资源 |
| 部署方式 | 一个镜像 | **两个镜像**，可分属不同主机 |
| 升级方式 | 前后端一起升 | **各自独立升级** |

后端默认编译为纯 API 服务，由 `web_assets_stub.go`（构建标签 `!embed_frontend`）提供空实现。构建不再依赖 `web/dist` 是否存在，Docker 镜像体积也小得多。

后端仓库**不再保留 `web/` 目录** —— 前端代码完全在独立仓库，二开仓库里一行前端源码都没有。

<details>
<summary><b>单体部署仍然保留</b>：用 <code>embed_frontend</code> 标签构建即可把前端嵌入回二进制</summary>

```bash
make build-embedded    # 构建 ../new-api-web 并把产物放进 web/dist 后编译
```

> `make build-embedded` 会在构建时自动创建 `web/dist`（构建产物，不纳入版本管理）。
> 如果只跑 `go build -tags embed_frontend` 而没有前端产物，embed 指令会因找不到
> `web/dist` 而报错 —— 这是预期行为。

</details>

### 补齐分离部署所需的配置能力

上游的后端假设前端与它同源，二开版本补齐了分离部署所需的全部能力：

| 能力 | 说明 |
| --- | --- |
| `FRONTEND_BASE_URL` | 浏览器访问后端根路径时 301 到前端；OAuth 回跳、密码重置链接、支付回跳统一使用它 |
| `API_BASE_URL` | 后端自身的公开地址，用于图片代理、视频内容、支付异步通知回调 |
| `CORS_ALLOWED_ORIGINS` | 精确 origin 白名单，允许携带 Cookie 跨域访问 API |
| `SESSION_COOKIE_SECURE` | 启用 Secure Refresh Cookie，以及 refresh/logout 的严格 Origin 校验 |
| `SESSION_COOKIE_TRUSTED_URL` | Secure 模式下可信的精确 HTTPS Origin 列表 |
| 纯 API 根路径响应 | 未配置 `FRONTEND_BASE_URL` 时，未知路径返回 JSON 404 而非 HTML，浏览器导航才 301 |

会话体系本身（Access Token + HttpOnly Refresh Cookie + 服务端登录会话控制面）沿用上游设计，未作改动。

### 服务器本地构建镜像

`docker-compose.yml` 默认从当前目录的 `Dockerfile` 构建镜像（`build: { context: . }`），不再拉取 Docker Hub 上的远程镜像。上传源码到服务器后，一条命令即可完成构建与启动：

```bash
docker compose up -d --build
```

配套脚本 `scripts/package-for-server.ps1`（Windows）/ `scripts/package-for-server.sh`（Linux/macOS）会把源码打包成一个压缩包，直接传到服务器解压即可。完整步骤见 [服务器部署指南](./docs/deployment/server-backend.md)。

---

## 快速开始

### 方式一：Docker Compose（推荐）

```bash
git clone https://github.com/Bitsea19/new-api-backend.git
cd new-api-backend

# 编辑 docker-compose.yml：改数据库和 Redis 密码，设置 SESSION_SECRET
docker compose up -d --build
docker compose logs -f new-api
```

后端本身不含界面，初始化完成后：

- **API 根路径**：`http://<服务器>:3000/api/status`
- **面板**：访问前端仓库部署出来的地址

### 方式二：本地开发

```bash
# 后端仓库根目录
go run .
```

```bash
# 前端仓库根目录（另开一个终端）
bun install
bun run dev
```

开发服务器会把 `/api`、`/v1`、`/v1beta`、`/mj` 和 `/pg` 代理到后端。打开 [http://localhost:5173](http://localhost:5173)。

> ⚠️ 对接**远程**后端开发时，注意不要把 `VITE_API_BASE_URL` 设成远程地址 —— 那会让请求变成跨域，
> 而后端的 Refresh Cookie 是 `HttpOnly` + `SameSite=Strict`，跨站带不过去，登录态会断。
> 保持它为空、只设置代理目标即可。详见[前端仓库的说明](https://github.com/Bitsea19/new-api-web)。

---

## 配置

### 存储

| 组件 | 可选方案 |
| --- | --- |
| 主数据库 | SQLite、MySQL ≥ 5.7.8、PostgreSQL ≥ 9.6 |
| 独立日志库 | 通过 `LOG_SQL_DSN` 配置，额外支持 ClickHouse |
| 缓存 | 可选 Redis 与内存缓存；多节点需要共享限流额度时使用共享 Redis |
| 容器平台 | Linux amd64 / arm64 |

### 环境变量

| 变量 | 用途 |
| --- | --- |
| `SQL_DSN` | 主数据库连接串；未设置时使用 SQLite |
| `LOG_SQL_DSN` | 可选的独立日志数据库连接串 |
| `REDIS_CONN_STRING` | Redis 连接串 |
| `SESSION_SECRET` | 持久保存的鉴权密钥，所有节点必须一致 |
| `CRYPTO_SECRET` | 默认使用 `SESSION_SECRET`；共享 Redis 的节点必须使用相同的有效值 |
| `FRONTEND_BASE_URL` | 前端面板的公开地址，用于浏览器导航重定向与各类回跳 |
| `API_BASE_URL` | 后端 API 的公开地址，用于图片代理、视频内容与支付异步通知 |
| `CORS_ALLOWED_ORIGINS` | 允许携带 Cookie 跨域访问 API 的精确 origin 白名单 |
| `SESSION_COOKIE_SECURE` | HTTPS 控制台设为 `true`，启用 Secure 刷新 Cookie 和严格的刷新／退出来源校验 |
| `SESSION_COOKIE_TRUSTED_URL` | Secure 模式必填：以逗号分隔的精确 HTTPS Origin，不含路径或通配符 |
| `TRUSTED_PROXIES` | 可信反向代理 IP/CIDR，或 `none`；按实际网络显式配置 |

> 完整配置见[环境变量示例](./.env.example)和[鉴权与登录会话说明](./docs/authentication.md)。
> 容器变量应通过 Compose 的 `environment` 或 `env_file` 注入 —— 只复制 `.env.example` 不会自动将变量传入容器。

---

## 部署

| 场景 | 文档 |
| --- | --- |
| 后端在服务器上本地构建镜像 | [服务器部署指南](./docs/deployment/server-backend.md) |
| 前后端分离的完整部署（域名、证书、反向代理） | [前端仓库的部署指南](https://github.com/Bitsea19/new-api-web/blob/main/deploy/DEPLOYMENT.md) |
| 鉴权、会话与安全配置 | [鉴权与登录会话说明](./docs/authentication.md) |
| 渠道额外设置 | [渠道额外设置说明](./docs/channel/other_setting.md) |
| 任务插件开发 | [Task Plugin API v1](./docs/plugin-api/v1.md) |

正式部署时使用 HTTPS，并配置反向代理支持流式响应与 WebSocket 升级。持久化并备份数据库和挂载数据。多节点必须共用主数据库和鉴权密钥。

---

## 核心能力

| 方向 | 可以做什么 |
| --- | --- |
| 模型接入 | 支持 OpenAI Chat Completions、Responses、Anthropic Messages 和 Gemini 协议，以及上游支持的流式输出、工具调用、推理与多模态输入 |
| 渠道调度 | 配置模型映射、渠道优先级与权重、失败重试、渠道亲和性和多密钥管理 |
| 用量与成本 | 管理额度、订阅套餐、用量日志、缓存计费，以及基于表达式的阶梯定价 |
| 访问控制 | 管理用户、分组、细粒度权限和 API Key 限制；支持 OAuth/OIDC、通行密钥、两步验证与登录会话管理 |
| 异步任务 | 通过 JavaScript 插件扩展图片、视频等任务 API，统一查询任务状态和获取产物 |

### 协议与接口

| 接口类型 | 常用入口 |
| --- | --- |
| OpenAI Chat / Responses | `POST /v1/chat/completions`、`POST /v1/responses` |
| Anthropic Messages | `POST /v1/messages` |
| Gemini | `POST /v1beta/models/{model}:generateContent`、`POST /v1beta/models/{model}:streamGenerateContent` |
| Realtime / Responses WebSocket | `GET /v1/realtime`、`GET /v1/responses`（WebSocket 升级） |
| 图片 / 音频 | `/v1/images/generations`、`/v1/images/edits`、`/v1/audio/speech`、`/v1/audio/transcriptions`、`/v1/audio/translations` |
| 向量 / 重排 | `POST /v1/embeddings`、`POST /v1/rerank` |
| 任务插件 | `POST /v1/tasks/{pluginKey}`、`GET /v1/tasks/{taskId}`，以及各插件声明的协议路由 |

[RelayKit](./relaykit/README.md) 提供上述四种文本协议之间的请求、响应和流式转换。实际可用能力取决于渠道、上游模型和转换路径。

### 目录结构

| 目录 | 职责 |
| --- | --- |
| `router/`、`middleware/`、`controller/` | HTTP 路由、访问校验与 API 处理 |
| `relay/` | 上游适配与请求调度 |
| `service/`、`model/` | 业务逻辑与持久化 |
| [relaykit/](./relaykit/README.md) | 可独立构建的协议 DTO 与转换 Go 模块 |
| [tokenkit/](./tokenkit/README.md) | 可独立构建的文本与图片 Token 计数 Go 模块 |
| [plugins/tasks/](./plugins/tasks/) | JavaScript 任务插件；编写方式与宿主边界见 [Task Plugin API v1](./docs/plugin-api/v1.md) |
| `electron/` | 桌面封装与打包 |
| `scripts/` | 服务器部署辅助脚本 |
| `docs/` | 项目文档 |

---

## 开发

后端使用 Go 和 Gin。Go 语言基线见 [go.mod](./go.mod)，容器构建工具链见 [Dockerfile](./Dockerfile)。

前端在独立仓库 [new-api-web](https://github.com/Bitsea19/new-api-web)，其技术栈为 React 19、TypeScript、Rsbuild、TanStack 与 Tailwind CSS 4，使用 Bun 管理依赖与脚本。

贡献前请阅读 [AGENTS.md](./AGENTS.md)。按改动范围执行检查：

| 范围 | 检查命令 |
| --- | --- |
| Go 模块 | `make test` |
| RelayKit / TokenKit | 在对应模块目录下 `GOWORK=off go build ./...` |
| 前端（在面板仓库） | `bun check -b`、`bun run lint`、`bun run test`、`bun run build` |

---

## 同步上游

本仓库是上游的 fork，需要持续跟进上游的修复与功能。

| 分支 / 远程 | 作用 |
| --- | --- |
| `origin`（Bitsea19/new-api-backend） | 本仓库的正式仓库，二开版本推送到这里 |
| `upstream`（QuantumNous/new-api） | 上游原始仓库，用于拉取上游改动 |
| `main` | 二开主分支，包含全部二开改动 |

### 为什么不能直接 merge

上游 `main` 分支**仍然带着完整的 `web/` 目录**，而本仓库早已把前端剥离到独立仓库，连 `web/` 目录本身都不保留。直接 `git merge upstream/main` 的实测结果：

| 冲突位置 | 数量 |
| --- | --- |
| 后端代码（`model/`、`relay/`、`middleware/` 等） | **0 个** ✅ |
| `web/` 目录（modify/delete 假冲突） | 18 个 |

也就是说，**直接 merge 不会破坏后端二开**，但会把上游的前端源码拖回来，造成大量假冲突和目录污染。

### 用同步脚本

同步脚本是**本地维护工具**，不纳入版本管理，因此不会出现在 GitHub 仓库里。它保存在你本地的 `更新/` 目录：

```
F:\我的开发\更新\
  sync-all.ps1        一次同步前后端两个仓库
  sync-web.ps1        只同步前端仓库
  sync-backend.ps1    只同步后端仓库
```

```bash
# 一次同步两个仓库
powershell -ExecutionPolicy Bypass -File F:\我的开发\更新\sync-all.ps1

# 只同步后端
powershell -ExecutionPolicy Bypass -File F:\我的开发\更新\sync-backend.ps1

# 预演，不实际改动
powershell -ExecutionPolicy Bypass -File F:\我的开发\更新\sync-backend.ps1 -DryRun
```

脚本会做标准三方合并（保住后端语义），再自动移除上游带来的 `web/`，最后报告冲突分布：

```
==> 冲突总数: 18
==> 其中 web/ 目录（将被自动剔除）: 18
==> 真正需要人工处理的: 0
```

合并后检查、提交、推送：

```bash
cd F:\我的开发\new-api-backend
git status
git commit -m "Merge upstream: 同步 QuantumNous/new-api 更新"
git push origin main
```

### 真冲突了怎么办

只有 `web/` **之外**的冲突才需要人工处理（上游和二开改了同一个后端文件）：

```bash
git checkout --ours <文件>    # 保留二开版本
git checkout --theirs <文件>  # 采用上游版本
git add <文件>
```

本仓库的二开集中在 `web_assets_stub.go`、`web_assets_embed.go`、`common/`、`setting/`、`Dockerfile`、`docker-compose.yml` 与 `docs/deployment/`，上游很少改动这些位置，因此基本不会产生真冲突。

> 💡 **前端同步方式不同**：面板仓库是整个 `web/` 目录的独立 fork，采用快照基线三方合并（取上游 web/ 的更新）。
> 两个仓库必须**分别同步**：上游改前端 → 同步前端仓库；上游改后端 → 同步后端仓库。
> 后端同步脚本不会把上游前端代码带进来，前端同步脚本也不会碰后端代码。
> 详见 [new-api-web 的同步说明](https://github.com/Bitsea19/new-api-web#%EF%B8%8F-%E5%90%8C%E6%AD%A5%E4%B8%8A%E6%B8%B8-quantumnousnew-api)。

---

## 文档索引

| 资源 | 入口 |
| --- | --- |
| 官方文档 | [使用指南](https://docs.newapi.ai/zh/docs) · [安装部署](https://docs.newapi.ai/zh/docs/installation) · [API 参考](https://docs.newapi.ai/zh/docs/api) |
| 前后端分离部署 | [服务器部署指南](./docs/deployment/server-backend.md) · [前端部署指南](https://github.com/Bitsea19/new-api-web/blob/main/deploy/DEPLOYMENT.md) |
| 鉴权与会话 | [鉴权与登录会话说明](./docs/authentication.md) |
| 项目导读 | [DeepWiki](https://deepwiki.com/QuantumNous/new-api) |
| 缺陷与功能建议 | [GitHub Issues](https://github.com/QuantumNous/new-api/issues)（上游） |

反馈问题时请附上版本、部署方式、复现步骤及脱敏日志。

---

## 合规声明

> [!IMPORTANT]
> - 本项目仅面向合法授权的 AI API 网关、组织内部鉴权、多模型管理、用量统计、成本核算和私有化部署场景。
> - 使用者必须合法取得上游 API Key、账号、模型服务或接口权限，并遵守上游服务条款及适用法律法规。
> - 使用者应确保其使用方式符合上游服务条款及适用法律法规。
> - 面向公众提供生成式人工智能服务时，使用者应遵守[《生成式人工智能服务管理暂行办法》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)等监管要求，自行完成所在司法辖区要求的备案、许可、内容安全、实名、日志留存、税务和上游授权等合规义务。

> [!WARNING]
> 将本项目作为面向公众的生成式 AI 服务或 API 转售服务运营时，使用者应先完成备案、内容安全、实名、日志留存、税务、支付和上游授权等合规义务。

---

## 许可证

本项目采用 [GNU Affero 通用公共许可证 v3.0 (AGPLv3)](./LICENSE) 授权。

根据 AGPLv3 第 7 条，本项目还适用[附加条款](./NOTICE)。修改版本必须在适当法律声明及界面中显著的关于、法律、页脚或署名位置保留作者署名 `Frontend design and development by New API contributors.`，并保留指向原项目 <https://github.com/QuantumNous/new-api> 的可见链接。

本项目为开源项目，在 [One API](https://github.com/songquanpeng/one-api)（MIT 许可证）的基础上进行二次开发，并在 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 的基础上完成前后端分离改造。

如果您所在的组织政策不允许使用 AGPLv3 许可的软件，或您希望规避 AGPLv3 的开源义务，请发送邮件至：[support@quantumnous.com](mailto:support@quantumnous.com)

署名及依赖声明见 [NOTICE](./NOTICE) 和[第三方许可证](./THIRD-PARTY-LICENSES.md)。

---

## 相关项目

| 项目 | 说明 |
| --- | --- |
| [new-api-web](https://github.com/Bitsea19/new-api-web) | **配套前端面板**，本二开版本的前端仓库 |
| [QuantumNous/new-api](https://github.com/QuantumNous/new-api) | **上游项目**，本仓库的二次开发来源 |
| [One API](https://github.com/songquanpeng/one-api) | 原版项目基础 |
| [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy) | Midjourney 接口支持 |

---

<div align="center">

### 💖 感谢使用 New API

如果这个项目对你有帮助，欢迎给它一个 ⭐️ Star！

**上游项目**：[QuantumNous/new-api](https://github.com/QuantumNous/new-api) • **[官方文档](https://docs.newapi.ai/zh/docs)**

<sub>Built with ❤️ by QuantumNous and contributors</sub>

</div>
