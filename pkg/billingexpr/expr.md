# 计费表达式体系（billingexpr）

## 设计哲学

**一个表达式，一份事实。** 一个表达式字符串完整定义一个模型的计费逻辑 —— 定价、阶梯条件、缓存／图片／音频区分、基于时间的折扣、感知请求的乘数 —— 全部写在一行里。没有散落各处的配置，没有隐式规则，没有魔法数字。

表达式是管理员与系统之间的计费契约。你写下什么就会被执行什么。系统的职责是忠实地求值它，而不是去解读它。

### 核心原则

1. **表达式是自包含的** —— 单凭表达式字符串就能确定计费。没有外部倍率表，没有隐式的补全乘数，没有隐藏的换算系数。给定相同的 Token 计数与请求上下文，同一个表达式永远产生相同的费用。

2. **变量是可选引入的** —— `p`（prompt）与 `c`（completion）是基础。缓存（`cr`、`cc`、`cc1h`、`cc12h`）、图片（`img`）与音频（`ai`、`ao`）变量是可选的。省略时，这些 Token 计入 `p`/`c` 并按其费率计价；省略的长效缓存写入保持其短效写入价格的固定倍数。系统会自动检测表达式使用了哪些变量（通过 AST 内省），并相应调整 Token 归一化。

3. **价格是真实价格** —— Token 系数是供应商公开的每百万 Token 美元价格。`p * 2.5` 表示每百万 prompt Token 2.50 美元；`fixed(0.01)` 表示每次请求 0.01 美元。不需要倍率换算，也不需要 `/2` 这类约定。

4. **与上游无关** —— 表达式无需知道上游 API 是 OpenAI 格式（`prompt_tokens` 含缓存）还是 Claude 格式（`input_tokens` 不含缓存）。系统会根据上游响应格式在求值前归一化 Token 计数。

5. **带版本** —— 表达式携带版本标签（`v1:`，省略时默认）。版本控制编译环境、Token 归一化与额度换算公式，从而在不破坏既有表达式的前提下支持未来的演进。

---

## 表达式语言

基于 [expr-lang/expr](https://github.com/expr-lang/expr)。表达式会被编译、缓存，并在运行时环境中求值。

### Token 变量

**输入侧变量：**

| 变量 | 含义 |
|------|------|
| `p` | 输入 Token 数（**计价用**）。**自动排除**表达式中单独计价的子类别（见下方说明） |
| `len` | 输入上下文总长度（**条件判断用**）。不受自动排除影响，始终反映完整输入长度。非 Claude：等于原始 `prompt_tokens`；Claude：等于文本输入 + 缓存读取 + 缓存创建 |
| `cr` | 缓存命中（读取）Token 数 |
| `cc` | 缓存创建 Token 数（Claude 5分钟 TTL / OpenAI 30分钟 TTL / 通用） |
| `cc1h` | 缓存创建 Token 数 —— 1小时 TTL（Claude 专用） |
| `cc12h` | 缓存创建 Token 数 —— 12小时 TTL（OpenAI 专用） |
| `img` | 图片输入 Token 数 |
| `img_cr` | 图片缓存读取 Token 数；只有表达式显式引用且上游提供有效缓存模态明细时才单独拆分 |
| `ai` | 音频输入 Token 数 |

**输出侧变量：**

| 变量 | 含义 |
|------|------|
| `c` | 输出 Token 数。**自动排除**表达式中单独计价的子类别（见下方说明） |
| `img_o` | 图片输出 Token 数 |
| `ao` | 音频输出 Token 数 |

#### `p` 与 `c` 的自动排除机制

`p` 与 `c` 是「兜底变量」—— 它们代表**所有没有被表达式单独定价的 Token**。系统会根据表达式实际使用了哪些变量，自动从 `p` / `c` 中减去对应的子类别 Token，避免重复计费。

**规则：如果表达式使用了某个子类别变量，对应的 Token 就从 `p` 或 `c` 中扣除；如果没使用，那些 Token 就留在 `p` 或 `c` 里按基础价格计费。**

> **重要：`len` 不受自动排除影响。** `len` 始终代表完整的输入上下文长度，不管表达式是否单独对缓存／图片／音频定价。因此**阶梯条件应使用 `len` 而非 `p`**，以避免缓存命中导致 `p` 降低而误判档位。

举例说明（假设上游返回的原始数据：`prompt_tokens=1000`，其中包含 200 cache read、100 image）：

| 表达式 | `p` 的值 | 说明 |
|--------|---------|------|
| `p * 3 + c * 15` | 1000 | 没用 `cr`/`img`，所以缓存和图片都包含在 `p` 里，全按 $3 计费 |
| `p * 3 + c * 15 + cr * 0.3` | 800 | 用了 `cr`，缓存 200 从 `p` 中扣除，按 $0.3 单独计费；图片仍在 `p` 里按 $3 计费 |
| `p * 3 + c * 15 + cr * 0.3 + img * 2` | 700 | 用了 `cr` 和 `img`，都从 `p` 中扣除，各自按自己的价格计费 |

输出侧同理（假设 `completion_tokens=500`，其中包含 100 audio output）：

| 表达式 | `c` 的值 | 说明 |
|--------|---------|------|
| `p * 3 + c * 15` | 500 | 没用 `ao`，音频输出包含在 `c` 里按 $15 计费 |
| `p * 3 + c * 15 + ao * 50` | 400 | 用了 `ao`，音频 100 从 `c` 中扣除按 $50 计费 |

长效缓存写入（Claude `cc1h`、OpenAI `cc12h`）没有单独计价时，按短效写入价格的固定倍数计费：
OpenAI 12 小时为 2 倍，Claude 1 小时为 6÷3.75 倍。短效写入价格是 `cc`，`cc` 也未计价时是 `p`。

#### 图片缓存与兼容性

OpenAI 兼容图片接口可报告
`usage.input_tokens_details.cached_tokens_details.{text_tokens,image_tokens,audio_tokens}`。
统一 DTO 保留各字段的缺失与显式零值；复制、用量合并及计费转换不得丢失这些信息。
当前公开 Images API 文档未明确缓存模态字段，兼容结构的合成测试不能替代真实上游缓存回包验证。

当表达式使用 `img_cr` 且明细有效时，`img_cr` 从 `cr` 和 `img` 中同时移出，
再按表达式引用的变量从 `p` 中扣除各互不重叠的类别。`len` 保持原始输入总长度。
例如总输入 1000、图片输入 600、缓存总量 300、图片缓存 200：
`p * 5 + cr * 1.25 + img * 8 + img_cr * 2 + c * 30` 对应
`p=300, cr=100, img=400, img_cr=200`；输出 100 时费用为 $0.008225。

缺少图片缓存明细时，`img_cr=0`，缓存总量继续进入 `cr`，沿用原有通用缓存计费，
不推测图片缓存比例。不引用 `img_cr` 的既有表达式保持原有归一化行为。
负数、超过总量或无法形成有效输入集合的明细会记录诊断并回退原有逻辑。
Anthropic 的独立缓存计数语义不采用这一 OpenAI 图片拆分。

图片缓存表达式的消费日志保存 `image_cache_tokens` 与 `billing_tokens`，后者由结算结果携带，
通过公共日志注入记录实际参与结算的计价用量，覆盖文本、audio 和 realtime 入口。
求值失败回退预扣时不伪造实际用量；固定价格和未引用 `img_cr` 的表达式不新增这些字段。
文本／图片日志的 `cache_tokens` 继续保留上游缓存总量。Images 的 `output_tokens` 是图片输出，
内置图片价格使用 `c`，无需上游额外报告 `output_tokens_details.image_tokens`。

### 旧定价转换

旧倍率和旧按次模式仍可运行，管理界面标记为弃用，新建定价默认使用表达式。
`POST /api/option/model_pricing/convert` 接收 `{model_name, pricing}`，其中 `pricing`
为完整的旧定价草稿，缺少的键继承运行时默认值。接口返回表达式及本次转换使用的生效定价快照，或具体不支持原因，不写数据库。
管理界面先用该快照显示左右对照预览，确认后才更新草稿；取消不改变原草稿。
保存仍使用现有 `PATCH /api/option/model_pricing` 及 `expected_version` 冲突检查。

倍率转换以 `ModelRatio * 1000000 / QuotaPerUnit` 得到 USD/百万 Token 的基础单价，
保留输出、缓存读取、适用的缓存创建和图片输入的生效倍率及显式零值。
常规按次价格转换为 `tier("request", fixed(price))`。旧价格字段继续保存，供切回旧模式使用。
转换后的预扣及舍入采用表达式规则，不保证旧模式每笔舍入结果完全一致。
迁移、已保存价格展示与草稿价格预览共用生效价格解析：输出使用模型族的硬编码兜底／强制倍率；
缺少配置的缓存读取、缓存创建、图片输入分别使用运行时的 1、1.25、1 倍。
图片倍率为 1 时，转换不生成独立 `img` 项，图片输入继续包含在 `p` 中按相同价格计费；
图片倍率为其他值（包括显式 0）时才生成 `img` 项。
缓存读取倍率为 1 时在价格预览中合并为输入价格；没有独立图片输入或缓存写入项时，转换同时省略 `cr`。
存在这些独立项目时，缓存计数可能交叠，输入余量的零下限使直接合并不再等价，因此保留原 `cr` 计算项而不单列同价价格。
基础输入价为 0 时可直接省略倍率为 1 的 `cr`；其他缓存读取倍率（包括显式 0）保留独立项。

省略 `cr` 时，Anthropic 用量中单独报告的缓存读取数会加回 `p`，而 `len` 保持完整上下文长度。
缓存创建按对外计费名和原始配置处理：名称包含 `claude`（不区分大小写）时保留 `cc` 和 `cc1h`，
其中 1h 价格沿用旧引擎的 `CreateCacheRatio * 6 / 3.75`；其他名称仅在草稿已配置 `CreateCacheRatio`
时生成通用 `cc`（包括显式 0），不生成 `cc1h`。这时名称包含 `gpt`（不区分大小写）的模型同时生成 `cc12h`，
价格沿用旧引擎的 `CreateCacheRatio * 2`。没有配置的通用 1.25 倍兜底不产生新计费项。
不按渠道映射或模型白名单扩展缓存类型，也不要求管理员另选类型。
定价快照条目、草稿预览响应和转换结果附带只读 `cache_write_mode`（`none` / `standard` / `claude_ttl` / `openai_ttl`），
用于统一展示。`POST /api/option/model_pricing/preview` 的响应统一为
`{success, message, data: {effective, cache_write_mode, billing_details}}`，调用者从 `data.effective`
读取生效价格；转换接口和快照条目的 JSON 结构不变。元信息不保存、不参与表达式执行。

未设置与显式 0 必须区分。草稿关闭某项覆盖后按保存后的回退规则解析，不能继续读取运行进程中尚未被保存替换的旧值。
图片和普通音频规则均可自动迁移，具体规则如下；任务插件、视频、Realtime，以及依赖上游 cost
反推缓存写入用量的 OpenRouter Claude 分支仍返回具体原因，不按渠道类型或未知端点一概拦截。

Gemini 音频输入使用旧结算实际采用的美元单价生成 `ai`，不乘普通输入倍率，因此普通输入价为零时
仍保留独立音频费用。普通音频的 `ai` 单价为基础输入价乘 AudioRatio，`ao` 再乘 AudioCompletionRatio，
缺省倍率按 1 处理，显式零保留。普通音频结算不使用缓存／图片倍率；模型同时配置这些价格时，
转换生成音频请求与纯文本请求两个分支。音频分支使用 `max(len - ai, 0)` 计量普通输入，避免纯文本
分支中引用的缓存变量改变音频费用。Gemini 文本结算仍保留缓存／音频交叠后的输入余量零下限。
只读 `billing_details` 为预览提供实际音频单价、图片数量和请求倍率规则，不成为新的持久化价格来源。

同一计费名在 Gemini／兼容音频入口存在相互冲突的单价时，
转换返回该具体冲突，不能静默选择其中一条规则覆盖其他入口。

图片按次价格转换为 `tier("image", fixed(price)) * image_count`，再追加旧尺寸、质量条件倍率。
DALL·E 尺寸／质量规则按原始请求模型名生成，模型映射不会额外引入这些规则。
OpenAI 已于 2026-05-12 下线 DALL·E 2/3；其校验、默认值和倍率保留在独立 legacy 文件，
仅用于历史配置及兼容上游。转换从 DTO 的旧计费逻辑获取倍率，不维护第二份价格表。
图片 token 定价不额外乘数量。所有金额和倍率固化在表达式中，不再叠加旧 OtherRatios。

### 图片数量

`image_count` 是独立于 Token 的计费数量。省略上下文时默认 1；提供的数量必须是 1 到
`dto.MaxImageN`（128）的整数。固定价格语法允许它作为乘数，仍禁止用 Token 乘固定价格。
图片入口只解析一次 provider 计费标量并随请求携带。数量来自顶层 `n`（`n=0` 兼容为 1）；
`parameters.n` 仍在入口校验范围，但不再覆盖计费数量。每次渠道重试或参数覆盖之后，再校验最终上游数量并在发送前补足预留，
图片钱包预留使用原子余额检查，不能通过增加数量形成欠费后继续提交。
`estimated_image_count` 保存本次发送前的数量；JSON/multipart 图片参数上下文只保留计费需要的标量，不保留图片或提示词内容。
结算使用独立的实际数量，不修改被冻结的 `param("n")`。正常结束
的 SSE 可以减少数量，客户端提前断开不能减少应收数量。最终消费日志记录 `image_count`。
阿里百炼图片模型（万相、Qwen-Image、Z-Image）由 alibaba 任务插件通过 `openai_image` 宿主协议承接：
其数量是任务用量事实 `u("image_count")`，提交时按请求数量预留，完成时按上游 `usage.image_count`
（Qwen-Image-3.0 为 `usage.output_image_count`）或按各 `choices[].message.content[]` / `results[]`
中的图片载荷数结算，任务消费日志同样记录 `image_count`。

> **注意：** 自动扣除针对 GPT/OpenAI 格式的 API（`prompt_tokens` 包含子类别）。Claude 格式的 API 不重复扣除缓存；未独立计价的缓存读取和缓存写入加回输入。系统根据上游返回格式自动处理。

### 内置函数

| 函数 | 签名 | 用途 |
|------|-----------|---------|
| `tier` | `tier(name, value) → float64` | 记录匹配到的定价档位；必须包裹费用表达式 |
| `fixed` | `fixed(amount) → float64` | 每次请求的美元价格，只作为 `tier(name, fixed(amount))` 中的完整价格使用 |
| `param` | `param(path) → any` | 读取请求体中的 JSON 路径（使用 ojson、gjson 路径语法） |
| `header` | `header(key) → string` | 读取请求头取值 |
| `has` | `has(source, substr) → bool` | 子串检查 |
| `hour` | `hour(tz) → int` | 时区中的当前小时（0-23） |
| `minute` | `minute(tz) → int` | 当前分钟（0-59） |
| `weekday` | `weekday(tz) → int` | 星期几（0=周日，6=周六） |
| `month` | `month(tz) → int` | 月份（1-12） |
| `day` | `day(tz) → int` | 几号（1-31） |
| `max` | `max(a, b) → float64` | 数学最大值 |
| `min` | `min(a, b) → float64` | 数学最小值 |
| `abs` | `abs(x) → float64` | 绝对值 |
| `ceil` | `ceil(x) → float64` | 向上取整 |
| `floor` | `floor(x) → float64` | 向下取整 |

### 表达式示例

```
# 简单固定定价
tier("base", p * 2.5 + c * 15 + cr * 0.25)

# 按请求条件定价，回退分支使用 Token 定价
len <= 32000
  ? tier("short", fixed(0.01))
  : tier("long", p * 2 + c * 8)

# 多阶梯（Claude Sonnet 风格）—— 阶梯条件用 len
len <= 200000
  ? tier("standard", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6)
  : tier("long_context", p * 6 + c * 22.5 + cr * 0.6 + cc * 7.5 + cc1h * 12)

# 图片模型（无独立缓存／音频定价 —— 这些 Token 留在 p/c 中）
tier("base", p * 2 + c * 8 + img * 2.5)

# 带音频的多模态
tier("base", p * 0.43 + c * 3.06 + img * 0.78 + ai * 3.81 + ao * 15.11)
```

### 按次固定价格

`tier("request", fixed(0.01))` 把所选叶子中的全部 Token 计费替换为每次成功 HTTP/SSE 请求 0.01 美元的基础价格。其他叶子仍可使用 Token 定价。
分组倍率与请求乘数继续生效；既有的工具附加费单独计算并照常相加。流式不会按每个分块收取固定费用。
Realtime 与任务用量表达式在提交上游或预留之前就会拒绝 `fixed()`。

金额必须是有限、非负的数字字面量，且其 v1 缩放值有限。显式的 `fixed(0)` 合法且保持免费。
使用 `fixed` 的表达式必须由一个条件定价树与下文描述的标准请求乘数因子构成，且每个叶子都包在 `tier()` 中。
在一个叶子里同时加入固定计费与 Token 计费、把多个阶梯相加，或用 Token 乘固定价格，都会被拒绝。
校验检查原始 AST，包括优化或短路会跳过的分支。不含 `fixed()` 的既有表达式保持其原有语法与行为。

`fixed(amount)` 在内部返回 `amount * 1,000,000`，以保持 v1 既有的额度换算与舍入。预扣用估算值匹配条件；
结算针对实际用量或已有的本地估算用量重新求值，并通过常规计费会话协调分支变化。
缺失上游用量时保留既有的估算路径；一个按次计价的实际分支即使 Token 为零也能计费。失败保留常规退款策略。
结算求值出错保留预留及其估算计费单位。

求值结果会暴露 `billing_unit`（`token` 或 `request`），以及按次计价叶子在乘数之前的美元 `fixed_price`。
预扣快照携带 `estimated_billing_unit` 与 `estimated_fixed_price`；消费日志把实际值追加到 `other`。
显式的零价格也会出现在这些字段中。较旧的快照与日志无需迁移。表达式仍是唯一的定价配置。

### 请求规则（追加在 `|||` 之后）

请求条件乘数以 `|||` 分隔符追加在表达式之后：

```
tier("base", p * 5 + c * 25)|||when(header("anthropic-beta") has "fast-mode") * 6
```

这些因子在最终表达式中以普通乘法的形式存储（例如 `(tier(...)) * (condition ? 6 : 1)`），并在同一个计费程序中运行。

### 请求规则追踪

编译期，引擎会以下列精确形态为三元因子插桩：

```
<request-probe condition> ? <numeric literal> : 1
```

条件必须至少引用一个请求探针（`param`、`header`、`hour`、`minute`、`weekday`、`month` 或 `day`）。
两个分支都必须是数字字面量，且回退值必须等于 `1`。其他条件表达式（包括 `(condition ? 2 : 1.5)`）会正常求值但不被追踪。
仅整数的因子使用保留整数类型的追踪回调，因此插桩不会改变需要整数操作数的表达式（例如 `%`）。
内部追踪回调名称是保留的，不能用在已存储的表达式中。

编译缓存为每个被插桩的节点存储规范条件与乘数。每次运行都以完整的已检测规则列表开始，全部标记为未匹配；
回调会把实际求值为真的规则标记为已匹配。被表达式短路跳过的规则保持未匹配。
这保持了表达式的数值结果不变，也避免每个请求都重新解析它。

结算会把实际运行中的追踪复制到消费日志里：

```json
{
  "request_rules": [
    { "cond": "param(\"service_tier\") == \"fast\"", "multiplier": 2, "matched": true }
  ]
}
```

用量日志界面把 `request_rules` 当作权威规则列表并直接渲染。它解析 `cond` 只是为了生成友好的标签，
当解析器无法识别该条件时回退到规范条件文本。没有日志上下文的定价页面继续解析已存储的表达式用于展示。

---

## 任务用量表达式

任务插件可以通过 `meta.usageSchema` 暴露经过校验的、供应商专属的计费事实。表达式用 `u("key")` 读取这些事实。
一个字面量键必须先由插件 schema 声明，表达式才能保存。单插件模型未变更的已存表达式及其未变更的供应商覆盖，
在插件收窄该模型的档案后仍可保存；任何编辑都必须遵循当前档案。

多个插件可以用不同的用量 schema 服务同一模型。执行渠道决定用哪个插件；客户端不选择计费供应商。
任务表达式的解析顺序如下：

1. `billing_setting.plugin_billing_expr["<pluginKey>::<clientModel>"]`
2. 同一插件对最终映射模型的覆盖（如存在）
3. 客户端模型的表达式（当其计费模式为 `tiered_expr`）
4. 最终映射模型的表达式（当客户端模型没有表达式计费模式）

插件覆盖始终使用任务用量语义，与模型的计费模式无关。这份扁平的覆盖 map 是一个独立选项，
绝不向模型名称或模型价格同步导出贡献复合键。既有价格不会被改写。计费快照冻结选定的表达式；
完成时使用该快照，即使之后供应商定价发生变化。

模型级表达式的保存会对每个候选选定的 schema 做冒烟测试，跳过在同一完整草稿中已有自己覆盖的候选。
供应商表达式的保存会校验插件／模型绑定以及该插件的 schema；错误会同时指明模型与插件。
对共享模型而言，生效表达式引用未声明的字面量 `u()` 键时，该供应商就是未配置的，
包括当前请求会跳过的分支中的引用。提交会以 400 `model_price_error` 拒绝；管理与公开定价会暴露未配置状态。
公开的模型级元数据仍是 key 升序中第一个插件的视角，供应商变体单独附加。

即使只剩一个供应商，已保存的供应商覆盖依然可见。管理端会把不可用的插件／模型绑定暴露为 `stale`，
并允许保留或移除其既有表达式。对这些表达式的修改仍然需要有效绑定；写入会比对锁定的数据库快照而非进程缓存。
不会静默清理任何覆盖。公开的供应商变体包含其生效的 `billing_mode`，因此继承的倍率／按次定价与未配置的用量表达式可以区分开。

服务不同模型族的插件可以声明
`meta.usageProfiles: [{models: ["image-model"], schema: {...}, examples: [...]}]`。
每个档案为其声明的模型提供完整的用量 schema 与可选的展示示例。它**替换**而不是合并插件默认的
`usageSchema` 与 `usageExamples`。一个模型最多属于一个档案，档案中的模型名必须使用 `meta.models` 中声明的拼写。
允许空的档案 schema；缺失或 null 的 schema 不允许。Token 单位的档案需要自己的示例，校验与默认值相同。

宿主在选择用量元数据之前先解析模型别名。模型级定价与表达式保存 API 使用声明的目标；
同一插件中一个有多个目标的歧义别名保留插件默认值。用量校验使用执行插件与最终上游模型。轮询
从每个已保存任务的模型中选择元数据，包括混合模型的批次。没有档案声明的上游名称（例如渠道映射到供应商端点 ID）
回退到面向客户端的模型的档案。未匹配的模型与没有档案的插件保留插件默认值。

档案不会引入请求体白名单，也不会移除旧版倍率扩展。数值型钩子事实保持一致的浮点表示用于表达式求值，
包括选定档案之外的字段。保存表达式时会用选定 schema 检查字面量用量键；已存储的表达式不会被迁移。
单插件提交保留既有的键校验行为；共享模型强制执行上面那条兼容性规则。既有的运行时错误与额度安全检查继续生效。
完成时继续在覆盖实测事实之后求值冻结的表达式；不会引入 schema 快照或数据库迁移。

这是插件 API 版本 1 的可选新增。新宿主可原样接受既有插件。旧宿主会拒绝新的元数据字段，
因此请在安装声明了档案的插件之前先升级宿主。

数值事实是声明的规范单位（`second`、`count`、`token` 或 `credit`）下的有限非负值；
枚举事实是来自已声明取值列表的精确字符串。schema 描述只是展示元数据，绝不影响求值。
`token` 是上游计费 Token 的宿主单位（例如 doubao 的 `usage.completion_tokens`）。
`credit` 是供应商资源包单位的宿主单位（例如 kling 的 `final_unit_deduction`）。
两者共用 int32 额度边界，而不受 3600 秒／128 计数限制。

任务用量计费有一条与 Token 计费刻意不同的换算规则：

```
task quota  = 表达式输出（美元） * QuotaPerUnit * groupRatio
token quota = 表达式输出（美元/百万 Token）/ 1,000,000 * QuotaPerUnit * groupRatio
```

换句话说，任务表达式返回的就是该请求的美元成本。例如 `u("seconds") * 0.4` 表示每秒 0.40 美元。
引擎语义不会把任务输出除以一百万。

可视化编辑器生成、且公开定价能够识别的，正是以下这些规范任务形态。
超出这些形态的表达式在原始模式下仍然有效，但会回退到特殊表达式展示：

```
# 扁平单位定价
tier("base", u("seconds") * 0.4)

# 枚举阶梯（条件可把枚举比较与 && 组合）
u("mode") == "pro"
  ? tier("pro", u("seconds") * 0.8)
  : tier("std", u("seconds") * 0.4)

# 可选常量加上多个数值用量项
tier("base", 0.1 + u("seconds") * 0.4 + u("clips") * 0.05)

# 上游 Token 叠加（doubao Seedance tokens）
# 编辑器接收每百万 Token 的美元输入，并输出 / 1000000 字面量。
# 引擎语义不变：表达式仍然返回美元。
tier("base", u("tokens") * 9.8 / 1000000)

# 供应商 credit 叠加（kling 资源包单位）
# 系数是真实的每 credit 美元价格；没有 /1M 缩放。
tier("base", u("units") * 0.14)
```

阶梯主体是一个可选的非负常量加上一个或多个 `u("<数值字段>") * <单价>` 项。
Token 单位的字段使用规范缩放形态 `u("<field>") * <每百万 Token 美元价> / 1000000`。
Credit、second 与 count 字段保持裸的 `u("<field>") * <单价>` 形态。
阶梯条件是已声明枚举字段与取值之间的相等判断，可选地用 `&&` 连接，
链式三元表达式遵循与 Token 阶梯相同的排序规则。数值区间阶梯不属于当前的规范形态。
`|||` 之后的请求规则保持正交，使用与 Token 表达式相同的语法。

保存之前，宿主会编译每个表达式，拒绝选定任务插件未声明的字面量 `u()` 键，并对用量向量做冒烟测试。
每个数值字段都会在 0、1 与其宿主单位上限（`second` 3600、`count` 128、`token`/`credit` int32 最大值）下被测试。
枚举取值按笛卡尔积测试。冒烟向量上限为 64，过大的枚举维度缩减为首尾取值。
每个求值结果都必须是有限且非负的。

提交时用请求推导出的用量事实对表达式求值，并把表达式与这些事实一起冻结在计费快照中。
完成时，宿主按键逐一把完成事实覆盖到冻结的提交事实上，因此实测值替换估算值，
而完成钩子省略的事实保留其提交时的值。随后再次对同一个表达式求值；
因此某个事实的变化可能产生结算差额与不同的匹配档位。求值失败保留预扣的扣费。

---

## 架构

### 数据流

```
前端编辑器 → 存储 → 预扣 → 结算 → 日志展示
```

### 1. 前端编辑器

**文件**：`new-api-web/src/pages/Setting/Ratio/components/TieredPricingEditor.jsx`

两种编辑模式：
- **可视化模式**：按变量填写价格、按阶梯填写条件。通过 `generateExprFromVisualConfig()` 生成表达式。
- **原始模式**：直接编辑表达式字符串。包含常见模型的预设模板。

编辑器输出一个计费表达式字符串和一个可选的请求规则表达式字符串。
两者在存储前通过 `combineBillingExpr(billingExpr, requestRuleExpr)` 合并。

### 2. 存储

**文件**：`setting/billing_setting/tiered_billing.go`

存储在数据库 `options` 表中的两张 option map：
- `ModelBillingMode`：`{ "model-name": "tiered_expr" }` —— 为模型启用阶梯计费
- `ModelBillingExpr`：`{ "model-name": "tier(\"base\", p * 2.5 + c * 15)" }` —— 表达式本身

保存时会校验表达式：
1. 通过 `billingexpr.CompileFromCache()` 编译 —— 语法检查
2. 用示例 Token 向量做冒烟测试 —— 确保结果非负

### 3. 预扣（额度估算）

**文件**：`relay/helper/price.go` → `modelPriceHelperTiered()`

请求到达且模型使用 `tiered_expr` 计费时：
1. 从 `billing_setting.GetBillingExpr()` 加载表达式
2. 构造 `RequestInput`（请求头 + 请求体），供 `param()` / `header()` 函数使用
3. 用本地估算的输入 Token（`P` 与 `Len`）和零输出 Token（`C=0`）运行表达式。
   客户端的输出上限不影响预留。如果关闭了请求 Token 估算，估算输入为零。
4. 把输出换算为额度：`rawCost / 1,000,000 * QuotaPerUnit`。
   对按 Token 计价的叶子，该预留再乘以 `quota_setting.pre_consume_multiplier`（默认 1；任意有限正十进制数）。
   该乘数不改变上下文长度档位选择、实际结算，也不影响按次计价的叶子。
   快照会冻结该乘数，供图片数量重试使用；分组重试复用乘过该系数的预留。没有乘数的旧快照保留系数 1。
5. 创建 `BillingSnapshot` 并存入 `RelayInfo`。表达式与请求状态保持冻结，供结算使用。
   自动分组重试会在下次上游尝试之前，从选中的分组刷新依赖分组的字段。
   如果首个免费分组跳过了预扣，而重试选中了付费分组，则在该次尝试之前创建计费会话。
   如果已有会话移动到更贵的分组，其预留在发送前提升到该分组的估算值；
   更便宜的分组则要等实际用量结算后才退款。

钱包信任绕过使用 `quota_setting.trust_quota_usd`（默认 10 美元；0 表示禁用绕过）。
钱包与任何受限 API Token 都必须超过该阈值。钱包仍必须能覆盖估算预留才会考虑绕过。
订阅与异步任务不使用该绕过。旧版 Token 倍率定价同样按乘数预留估算的输入成本，
但不再有原先的 `PreConsumedQuota` 下限或输出 Token 估算。任务用量定价不变。

### 4. 结算（实际计费）

**文件**：`service/tiered_settle.go`、`pkg/billingexpr/settle.go`

上游响应返回并带上实际 Token 用量之后：

1. `BuildTieredTokenParams(usage, isClaudeUsageSemantic, usedVars)`：
   - 从 `dto.Usage` 读取实际 Token 计数
   - 对 GPT 格式 API（`prompt_tokens` 含全部内容）：**只有当**表达式使用了相应变量时
     （通过对已编译表达式的 AST 内省检测）才从 P/C 中减去子类别
   - 对 Claude 格式 API：表达式未涵盖的缓存读取与写入会加回输入；单独定价的缓存仍保持独立

2. `TryTieredSettle(relayInfo, params)`：
   - 使用捕获的 `BillingSnapshot`，其依赖分组的字段已从最终选中的分组刷新
   - 用实际 Token 计数重新运行表达式
   - 通过 `quotaConversion()` 换算（按版本分派）
   - 返回实际额度

### 5. 日志展示

**文件**：`service/log_info_generate.go`、`new-api-web/src/helpers/render.jsx`

后端：`InjectTieredBillingInfo()` 把 `billing_mode`、`expr_b64`（base64 表达式）、`matched_tier`
以及结构化的 `request_rules` 追踪列表加入日志的 `other` JSON。

前端：检测到 `billing_mode === "tiered_expr"` 时，解码 `expr_b64`，通过共享的 `parseTiersFromExpr()` 解析阶梯，
并在存在时从 `request_rules` 渲染请求乘数。没有日志追踪时，回退为解析已存储的表达式。

---

## 关键设计决策

### 通过 AST 内省实现 Token 归一化

不同的上游 API 报告 `prompt_tokens` 的方式不同：
- **OpenAI/GPT**：`prompt_tokens` = 总量（文本 + 缓存 + 图片 + 音频）
- **Claude**：`input_tokens` = 仅文本（缓存单独报告）

系统把 `p` 归一化为「未被单独定价的 Token」：**只有当表达式引用了相应子类别时**才减去它们。
这是通过遍历已编译 AST 查找 `IdentifierNode` 引用来确定的 —— 首次编译之后没有运行时开销（已缓存）。

示例：`p * 2.5 + c * 15 + cr * 0.25`
- 表达式使用了 `cr` → 缓存读取 Token 从 `p` 中扣除
- 表达式没有使用 `img` → 图片 Token 留在 `p` 中，按 $2.50 计价

### `len` —— 上下文长度变量

`len` 表示输入上下文总长度，专为**阶梯条件求值**而设计（例如 `len <= 200000 ? ...`）。
与 `p` 不同，`len` 绝不会被子类别排除机制削减。

**计算规则：**
- **非 Claude（GPT/OpenAI 格式）**：`len = prompt_tokens`（上游响应中的原始总量）
- **Claude 格式**：`len = input_tokens + cache_read_tokens + cache_creation_tokens`
  （由于 Claude 的 `input_tokens` 只含文本，必须把缓存加回才能反映完整上下文长度）

这确保大量使用缓存不会导致阶梯条件错误地落到更低的档位。
例如某请求总上下文 300K 但其中 250K 已缓存，扣掉缓存后的 `p` 只有 50K（标准档），
而 `len` 正确地报告 300K（长上下文档）。

### 额度换算

表达式系数是美元/百万 Token。换算为内部额度：

```
quota = exprOutput / 1,000,000 * QuotaPerUnit * groupRatio
```

这与按次计费模式一致：`quota = modelPrice * QuotaPerUnit * groupRatio`。

### 表达式版本管理

表达式可以携带版本前缀：`v1:tier(...)`。无前缀即 v1。

版本控制：
- 编译环境（可用变量与函数）
- Token 归一化逻辑
- 额度换算公式

这让未来的演进不会破坏既有表达式。

---

## 文件索引

| 层次 | 文件 |
|-------|-------|
| 表达式引擎 | `pkg/billingexpr/compile.go`、`run.go`、`settle.go`、`round.go`、`types.go` |
| 存储 | `setting/billing_setting/tiered_billing.go` |
| 预扣 | `relay/helper/price.go`、`relay/helper/billing_expr_request.go` |
| 结算 | `service/tiered_settle.go`、`service/quota.go` |
| 日志注入 | `service/log_info_generate.go` |
| 前端编辑器 | `new-api-web/src/pages/Setting/Ratio/components/TieredPricingEditor.jsx` |
| 前端展示 | `new-api-web/src/helpers/render.jsx`、`new-api-web/src/helpers/utils.jsx` |
| 模型详情 | `new-api-web/src/components/table/model-pricing/modal/components/DynamicPricingBreakdown.jsx` |
| 日志展示 | `new-api-web/src/hooks/usage-logs/useUsageLogsData.jsx`、`new-api-web/src/components/table/usage-logs/UsageLogsColumnDefs.jsx` |
