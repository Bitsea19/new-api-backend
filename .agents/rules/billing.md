# 计费规则

Go 后端中额度、定价、预扣费、结算、退款与用量数量相关代码的强制约定。`AGENTS.md` 定义了何时算「触及计费」（见 **计费规则（强制阅读门禁）**）；任何匹配的任务都必须在规划、编码或审查前完整读取本文件。不匹配任何触发条件的任务可以跳过。本文件的措辞与 `AGENTS.md` 一致，权重相同。

**计费表达式体系：** 处理阶梯／动态计费（基于表达式的定价）时，必须先读 `pkg/billingexpr/expr.md`。它记录了设计哲学、表达式语言、完整架构、Token 归一化规则、额度换算与表达式版本管理。所有计费表达式的改动都必须遵循该文档。

**内置模型定价：** 新增的内置模型价格必须以自包含的计费表达式形式定义在 `setting/billing_setting/builtin_billing.go` 中，使用真实的每百万 Token 美元价格。不要把新的内置价格加入旧的模型／补全／缓存倍率表。保留管理员显式设置的定价覆盖。只有在明确要求时才迁移既有旧价格。核对已发布的价格，并覆盖适用的上下文长度阈值与缓存类别。

**计费安全不变量：** 额度／计费代码绝不能因算术溢出或未校验输入而产生负计费（返还额度）。要做纵深防御：

- 每一个会变成计费乘数的用户可控数量（图片 `n`、视频 `seconds`／`duration`、分辨率／质量倍率、批量数量），在进入额度计算之前都必须被限定范围。越界值在请求校验阶段以 400 拒绝。现有边界：图片生成数量用 `dto.MaxImageN`，任务视频时长用 `relaycommon.MaxTaskDurationSeconds`，各中转格式（OpenAI、Claude、Gemini、Responses）的 `max_tokens` 系列字段用 `maxTokensLimit`（`relay/helper/valid_request.go`）。复用这些常量，不要为同一概念另起新的临时限制。新增中转格式或请求 DTO 时，从第一天起就在其校验器中限定 max-tokens 与计数字段。
- 警惕绕过校验的路径：透传字段（例如 `Extra["parameters"]`）、任务 `metadata` map 与 multipart 表单字段，都能绕过标准 DTO 校验把同样的数量带进来。任何从这类路径读取乘数的适配器都必须在本地强制执行同样的边界（或钳制）。任务插件自行校验自己的请求：宿主在额度计算前会限制插件返回的每一个用量事实，且不解释请求成员。插件必须从它实际发往上游的请求推导事实，包括从 `metadata` 或其他透传 map 合并进来的值，并限定任何被它转成 Token 或额度事实的请求数量；宿主会封顶 `second` 与 `count` 事实，但只对 `token` 与 `credit` 事实在 int32 额度边界处饱和。转发时长为 -1（由供应商决定长度）的插件，必须预留该模型的最长时长，并按供应商上报的用量结算。
- 从媒体元数据解析出的时长同样受用户／上游控制：音频文件头（转录 Token 计数、TTS 响应时长）和上游扣费数字（例如 Kling 的 `FinalUnitDeduction`）都可能声称离谱的数值。转成 Token 计数之前必须做饱和转换。
- 绝不要用裸类型转换把计算出的额度或 Token 计数转成 `int`，例如 `int(float64(quota) * ratio)`、对无界输入做 `int(math.Round(...))`、或 `int(decimal.IntPart())`。所有额度舍入／换算集中在 `common/quota_math.go`，使用其中的辅助函数：浮点乘积用 `common.QuotaFromFloat`（截断），需要舍入时用 `common.QuotaRound`（远离零舍入），十进制乘积用 `common.QuotaFromDecimal`。`billingexpr.QuotaRound` 委托给 `common.QuotaRound`。不要重新引入本地的换算辅助函数或裸转换。单次请求的饱和保持在 int32 边界，使批量累加无法逼近 64 位回绕；钱包／充值换算使用 `common.WalletQuotaFromDecimalStrict`，边界为 JavaScript 安全的 `common.MaxWalletQuota`。每一次钳制／NaN 回退都通过 `common.SysError` 记录日志。
- 饱和事件同样要可审计：每个辅助函数都有一个 `*Checked` 变体（`common.QuotaFromFloatChecked` / `QuotaRoundChecked` / `QuotaFromDecimalChecked`），发生钳制时额外返回 `*common.QuotaClamp`。计算出费用的计费路径要把钳制记录挂到 `relayInfo.QuotaClamp`（或透传到任务结算），并在写入消费／任务日志前调用 `attachQuotaSaturation`（位于 `service/log_info_generate.go`），它把标记嵌套到日志的 `other.admin_info.quota_saturation` 下，并发出带请求关联的 `logger.LogWarn`。嵌套在 `admin_info` 下可顺带获得「仅管理员可见」的效果（非管理员的日志视图会剥离 `admin_info`）。新增计费路径时，使用 `*Checked` 变体并以同样方式暴露钳制，使异常在管理端日志界面与后端日志中都保持可审计。
- 乘数 map 一律走 `types.PriceData.AddOtherRatio`，它会拒绝非正数、NaN 与 +Inf 的倍率。不要直接写 `PriceData.OtherRatios`，也不要削弱这些防护。
- 预扣费与结算／差额都必须安全：饱和后的超额额度必须在预扣费阶段以「额度不足」失败，绝不能静默回绕。新增计费路径（新中转格式、新任务平台、新调整钩子）时，追踪完整链路 —— 校验 → EstimateBilling/OtherRatios → 额度换算 → 预扣费 → 结算／退款 —— 确认每一步都保持了这些不变量。
- 解析为无符号类型（`*uint`）的字段会接受极大的正数 JSON（例如 `18446744073686646784`，即一个回绕的负数）；`>= 0` 检查并不足够，必须有上界。
- 这些不变量的回归测试应与它们所保护的边界放在一起（请求校验器、换算辅助函数）。期望风格参见 `relay/helper/openai_image_request_test.go`、`relay/common/relay_utils_test.go` 与 `common/quota_math_test.go`。

**上游图片数量结算：** 计费图片数只能通过 HTTP 图片中转上的 `RelayInfo.UpdateImageCount`，或任务插件完成时返回的 `image_count` 用量事实来下调或上调；为零或越界的数量会保留原先预留的请求数量。数量必须从图片负载推导，绝不能从 JSON 结构推导：

- 不要用 gjson 的 `data.#`、`len(data)` 或任何元素计数来统计 `data`。OpenAI 兼容渠道背后的供应商会返回非标准形状：`data` 是单个对象、一张图片被拆成 `url` 项与 `b64_json` 项、或各项只带 `revised_prompt`。
- 对象形状的 `data`，当且仅当它有非空的 `url` 或 `b64_json` 时算一张图，否则为零。对数组，数量等于 `max(带非空 url 的项数, 带非空 b64_json 的项数)`：标准响应只用一种 `response_format`，因此该值等于项数；拆分的图片只计一次；没有负载的项不计费。参考实现是 `relay/channel/openai/relay_image.go` 中的 `openaiImageResponseCount`；任何转发或重建 OpenAI 格式图片响应的渠道都应复用它（或同样的规则）。xAI 适配器已经通过 `OpenaiImageHandler` 继承了它。
- 把 OpenAI 格式的 JSON 负载重新以 SSE 发出时，每个图片负载要转发一个 `image_generation.completed` 事件，使用同样的形状处理（对象 → 一个事件，没有负载的项丢弃）。客户端绝不能只收到 `[DONE]` 却仍被收取图片费用。
- 真实的 SSE 流统计 `image_generation.completed` / `image_edit.completed` 事件，只有在上游已完成（`done` / `eof`）时才采信更小的数量；客户端中止不能把计费量降到请求数量之下。
- 阿里（百炼）图片模型通过 `alibaba` 任务插件以 `openai_image` 宿主协议提供服务。它的 `imageUsage` 先读 `usage.image_count`，再读 `usage.output_image_count`（Qwen-Image-3.0），否则统计摊平到每个 `choices[].message.content[]` 条目与 `results[]` 后的图片负载数量；没有负载的部分不计费，而上游给出的小数、负数、超大值或与负载不一致的数量都会保留原先预留的请求数量。任何拥有自己 usage 字段的其他渠道都适用同样的顺序：先采信有界的上游数量，再统计负载。从带类型结构体重建响应的渠道统计类型化 slice；它们都不能读 `data.#`。
- 对象形状、拆分形状与无负载形状的回归用例属于 `relay/channel/openai/image_stream_test.go`（固定价格、阶梯表达式与 JSON-as-SSE 转发表）。扩展这些表，不要新增文件。
