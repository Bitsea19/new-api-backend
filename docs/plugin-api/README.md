# 任务插件 API v1

任务插件是单文件的同步 ECMAScript 模块。插件契约当前尚未正式发布；[`v1.schema.json`](./v1.schema.json) 与 [`v1.d.ts`](./v1.d.ts) 是权威的 v1 契约。

## 契约与生命周期

每个插件都导出 `meta`、`buildSubmitRequest`、`parseSubmitResponse` 与 `parseTaskResult`。`per_task` 插件还导出 `buildQueryRequest`；`batch` 插件导出 `buildBatchQueryRequest` 与 `parseBatchResult`。`meta.author.name` 必填；`meta.author.url` 存在时必须是绝对 HTTP(S) URL。这是自行声明的署名；未来市场的「已验证发布者身份」是宿主单独维护的记录。

插件可以声明带鉴权的供应商原生 `meta.routes`，并通过 `meta.protocols` 占用宿主的协议名。提交与动态路由要指定一个 `native` 解码器与呈现器；查询路由只指定呈现器。协议绑定由宿主注册表一次性注册，协议解码器会收到宿主解析后的 `body` 联合类型以及被固定的模型。共享的协议钩子是同步变换；Go 负责连接与线格式分帧。

宿主选择渠道、调用请求构造钩子、针对渠道主机校验返回的 URL、执行 HTTP，并把解码后的响应交给对应的解析钩子。它掌管持久化、重试、轮询、计费与结算。插件只做数据变换并上报用量事实。函数签名见 [v1.d.ts](./v1.d.ts)，机器可读的结构见 [v1.schema.json](./v1.schema.json)。

暴露任务产物的插件要同时导出 `listArtifacts(task)` 与 `buildContentRequest(ctx)`。产物只在显式读取时从持久化的 `Task.Data` 中投影出来，绝不作为第二份事实来源单独存储。列表只包含稳定的 `key`、`type` 与可选的 `mimeType`。内容钩子收到选中的 key、原始的已解码任务数据、显式的私有上游任务 id、产出插件版本、渠道鉴权，以及一组安全的客户端 Range／条件请求头子集。它的 URL 与请求头只在那次代理请求中存在。

当一条 Responses 观测到达持久化的 `SUCCESS` 时，宿主还会运行被固定插件的 `listArtifacts`，并向 `renderEvents` 或 `renderFinal` 注入一份只读的 `ctx.artifacts[key] = {key, type, mimeType?, url}` map。每个 `url` 都是长期有效的宿主签名能力 URL，绝不是 `Task.Data` 里的供应商 URL。非终态与失败的任务不会收到产物 map。能力 URL 的构造或渲染失败只会让那一条 Responses 观测失败，无法改变任务、计费结算或退款。绝对 URL 使用 `TaskPublicAddress`，仅回退到 `ServerAddress`；多节点部署必须共用有效的 `CRYPTO_SECRET`。面板的产物读取直接返回各自的 `content_url`（或旧版的 `legacy_content_url`），不做临时 URL 交换。能力的生成与校验是无状态的且没有有效期；校验之后，宿主仍会加载提供该产物所需的任务、属主与插件。轮换 `CRYPTO_SECRET` 会让已签发的 URL 失效。`access` 查询参数在请求日志中会被脱敏。

部署边界与并发相关的环境变量见 [v1.md](./v1.md#通用任务管理接口)。

宿主要求 `protocols.openai_video.render` 返回一个 JSON 对象，并保留供应商的扩展字段，包括输出 URL 与元数据。插件可以返回 `task.data` 以暴露最近一次持久化的上游快照。宿主仍会覆写 `id`、`object`、`model`、`status`、`progress`、`created_at` 与 `completed_at`，并移除旧版的 `task_id`。这是字段级投影，不是逐字节的 HTTP 转发：已知的私有任务 id 会在渲染前被替换，轮询可能对内联视频数据脱敏，而获取操作读取的是最近保存的快照，而不是发起新的上游请求。产物内容端点仍可用于代理下载。

带供应商鉴权的内容 URL 必须使用渠道基础主机，或插件声明的 `meta.allowedHosts` 条目。公共动态 CDN URL 可以改为设置 `credentialless: true`；此时宿主只允许 GET/HEAD，不允许插件提供的请求头或请求体，并对初始 URL 及每一次重定向执行 SSRF 检查。

注册表发布是按代次原子的。一个请求在其整个生命周期内固定一个插件代次，而后台轮询可能使用更新的活跃插件版本。新版本必须继续能解析在途任务的响应。Root 管理员可以通过 `GET /api/plugin/task/runtime/status` 查看本节点状态。响应包含节点本地的代次、活跃数据库覆盖的确定性修订号、最近一次重建的结果，以及插件级的编译或路由错误。代次编号是节点本地的；诊断多节点间的发布滞后时要比较数据库修订号。如果数据库快照暂时不可用，该端点会继续以 `database_error` 字段提供节点本地状态与最近已知的修订号。

要现场排查，可以以 `DEBUG=true` 启动进程，并按 `task_plugin` 过滤日志。插件注册表、路由、端点归属、渠道选择、提交持久性、轮询适配器与协议观测都会输出安全的键值生命周期事件。请求上下文事件带请求 id；计划任务、后台任务与无上下文的工作标记为 `SYSTEM`。插件的 `console.log` 输出在 DEBUG 模式下也会被转发。钩子期输出带插件 key／版本前缀；模块初始化期的输出在首次上传校验期间可能没有身份标识。不要在插件代码中打印凭据、请求头、请求体、上游负载或私有 URL；自由格式的控制台输出无法被宿主脱敏。

## 夹具与试运行

一个夹具用例形如 `{name?, hook, member?, args, expected?, expectedError?}`。为每个导出的钩子、它的主要错误分支、批量行为、渲染器、用量与内容请求保留确定性用例。在本地运行夹具：

```sh
new-api plugin lint plugin.js
new-api plugin test plugin.js --fixture golden.json
```

Root 管理员可以打开插件详情页的沙盒标签，选择一个钩子并提交一份 `args` JSON 数组。`POST /api/plugin/task/:key/dryrun` 会在临时注册表中编译当前数据库源码或工厂源码，并且只调用那一个同步函数。试运行绝不会执行请求描述符，因此绝不会联系上游服务。

## 上传与发布

从仅 root 可见的任务插件页面，或通过 `POST /api/plugin/task` 附带 `{"source":"...","remark":"..."}` 上传。服务端会编译模块、校验 v1 元数据与必需导出，并在保存前拒绝无效源码。请使用语义化插件版本。用不同的源码复用同一个 key／版本会被拒绝；通过管理页面激活或回滚某个已存版本。

要接入第三方平台，创建一个 `Task Plugin` 类型的渠道，选择插件 key，填写明确的基础 URL，并配置模型。客户端可以使用该插件声明的原生路由。通用的管理接口仍然是 `POST /v1/tasks/:pluginKey`、`GET /v1/tasks/:taskId` 与 `GET /v1/tasks/:taskId/artifacts`，以及 `GET|HEAD /v1/tasks/:taskId/artifacts/:key/content`。

## 安全边界

插件没有全局 `fetch`、文件系统、`require`、import、异步函数、从字符串编译的代码（`eval` 与 `Function` 构造器会抛出 `EvalError`）或环境访问。少数驱动钩子可以用 [`utils.fetch`](./v1.md#用-utilsfetch-读取上游数据) 读取上游数据：只允许 `GET` 或 `HEAD`，且目标主机限于请求描述符可以寻址的主机。宿主限制执行时间、并发数、输入大小、允许的请求主机，并在 JavaScript 之外解析 OAuth 凭据。multipart 文件只以不透明引用的形式进入 JavaScript。

这不是硬性的内存隔离边界。插件能看到当前请求所需的数据，也能影响一次带鉴权的上游请求。上传插件是一项管理员级别的信任决策，等同于配置一份渠道凭据。激活前请审阅源码与版本 diff。绝不要仅仅因为插件能编译就运行不受信任的插件。

用量钩子可以返回秒数、分辨率或上游单位等事实，但绝不能计算价格或尝试做额度结算。所有定价与计费换算都由宿主掌管并封顶。
