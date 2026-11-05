---
name: i18n-translate
description: >-
  为本项目补全并维护前端 i18n 翻译。涵盖查找缺失的翻译键、
  检测未翻译条目，以及为所有受支持的语言环境（en、zh、zh-TW、fr、ja、ru、vi）
  添加翻译。凡涉及前端语言文件、缺失翻译键、未翻译的 UI 文本、
  `t(...)` 键、`useTranslation()`、静态 i18n 键、按钮／标签／
  toast／对话框／占位符／校验文案，或添加／修正哪怕只有一个
  i18n 键的任务，都应使用本技能。当审查结论提到缺少 i18n、当新的 UI 文本
  需要翻译，或当用户要求添加翻译、修复 i18n 或补全缺失翻译时，使用本技能。
  在翻译、添加语言键或编辑前端 i18n 文件之前，务必先加载并遵循本技能。
---

# 前端 i18n 翻译工作流

## 强制前置检查

- 在任何前端 i18n 工作之前通读整个 `SKILL.md`，包括只修一个键的情况。
- 编辑语言文件之前，确认源文本来自 `t(...)` 键、`en.json`、既有 UI 文案，或用户明确要求的新 UI 字符串。
- 只用用户对话来理解任务目标。不要把对话文本、审查措辞或任务描述直接写进语言文件。
- 翻译每个键之前，从代码与语言环境上下文重新思考它的界面含义，而不是把周围的对话当作翻译来源。

### 硬性约束：语言文件写入必须走脚本

- 你绝不能直接用文本编辑工具（StrReplace、Write、查找替换、手工改 JSON 等）编辑
  `new-api-web/src/i18n/locales/*.json`。即使只改一个键也不行。
- 所有语言文件写入都必须经过 `add-missing-keys.mjs` 脚本，随后运行 `bun run i18n:sync`。
  该脚本是添加或修改语言文件取值的唯一被认可的方式。
- 为什么这是强制的，而非可选的：
  - 手工编辑很容易漏掉七个语言环境（`en`、`zh`、`zh-TW`、`fr`、`ja`、`ru`、`vi`）中的一个或多个，
    导致某些语言缺少键。
  - 手工编辑会破坏要求的键字母序，还会引入 JSON 语法错误（尾随逗号、引号不匹配）。
  - 脚本会原子地写入全部七个文件并保持一致的排序，语言集合因此在结构上始终同步。
- 脚本不会替你做翻译。你仍然必须为每个语言环境推理出文案并填写脚本的 `newKeys` 对象；
  脚本只负责插入、排序与写入。不要因为「反正思考过程是一样的」就跳过脚本。

## 适用范围清单

编辑文件之前，只要任务涉及以下内容，就按本技能处理：

- `i18n`、翻译、语言文件、语言包、缺失键或未翻译文本
- `t('...')`、`useTranslation()`、`static-keys.ts` 或 `locales/*.json`
- 按钮、标签、toast、对话框、占位符、校验信息、描述或表格／空状态中的 UI 文案
- 关于缺少 i18n 键的审查结论

不要因为「只是修一个键」就跳过这个工作流。

## 概览

- 语言文件：`new-api-web/src/i18n/locales/{en,zh,zh-TW,fr,ja,ru,vi}.json`
- 格式：`"translation"` 键下的扁平 JSON，键为英文源文本
- 基准语言：`en.json`（键最多），回退：`zh`（中文）
- 同步脚本：`bun run i18n:sync`（在 `new-api-web/` 下执行）
- 所有 `t()` 调用都必须在每个语言文件中有对应的键

## 小修复路径

针对单个已知缺失的键（仍然只能走脚本，不能直接改 JSON）：

1. 在调用处确认确切的键，并验证它在所有语言文件中都不存在。
2. 通过 `add-missing-keys.mjs` 添加该键，为每个受支持的语言环境填写其 `newKeys` 对象：
   `en`、`zh`、`zh-TW`、`fr`、`ja`、`ru`、`vi`。即使只有一个键也要走脚本；不要手工改 JSON。
3. 脚本会保留扁平的 `"translation"` 对象，并自动保持键按字母序排列。
4. 针对该键在代码与语言文件中做一次定向检索。
5. 运行 `bun run i18n:sync` 归一化文件顺序。这一步是强制的，不是可选的。

## 工作流程

### 第 1 步：运行同步并阅读报告

```bash
cd new-api-web && bun run i18n:sync
```

阅读 `new-api-web/src/i18n/locales/_reports/_sync-report.json` 查看各语言环境的状态
（missingCount、extrasCount、untranslatedCount）。

### 第 2 步：查找缺失键（代码中使用了但语言文件中没有）

创建并运行 `new-api-web/scripts/find-missing-keys.mjs`：

```javascript
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')
const SRC_DIR = path.resolve('src')

const en = JSON.parse(await fs.readFile(path.join(LOCALES_DIR, 'en.json'), 'utf8'))
const enKeys = new Set(Object.keys(en.translation))

const tCallRegex = /\bt\(\s*['"`]([^'"`\n]+?)['"`]\s*[,)]/g
const tCallMultilineRegex = /\bt\(\s*['"`]([^'"`]+?)['"`]\s*\)/g

async function walkDir(dir) {
  const files = []
  const entries = await fs.readdir(dir, { withFileTypes: true })
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (['node_modules', '.git', 'locales', '_reports', '_extras'].includes(entry.name)) continue
      files.push(...(await walkDir(fullPath)))
    } else if (/\.(tsx?|jsx?)$/.test(entry.name)) {
      files.push(fullPath)
    }
  }
  return files
}

const files = await walkDir(SRC_DIR)
const missingKeys = new Map()

for (const file of files) {
  const content = await fs.readFile(file, 'utf8')
  const relPath = path.relative(SRC_DIR, file)
  for (const regex of [tCallRegex, tCallMultilineRegex]) {
    regex.lastIndex = 0
    let match
    while ((match = regex.exec(content)) !== null) {
      const key = match[1]
      if (key.startsWith('{{') || key.includes('${')) continue
      if (!enKeys.has(key)) {
        if (!missingKeys.has(key)) missingKeys.set(key, [])
        missingKeys.get(key).push(relPath)
      }
    }
  }
}

if (missingKeys.size === 0) {
  console.log('All t() keys found in en.json!')
} else {
  console.log(`Found ${missingKeys.size} missing keys:\n`)
  for (const [key, files] of [...missingKeys.entries()].sort(([a], [b]) => a.localeCompare(b))) {
    console.log(`  "${key}"`)
    for (const f of [...new Set(files)]) console.log(`    -> ${f}`)
  }
}
```

### 第 3 步：查找未翻译条目（取值与英文相同）

创建并运行 `new-api-web/scripts/find-untranslated.mjs`：

```javascript
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')
const en = JSON.parse(await fs.readFile(path.join(LOCALES_DIR, 'en.json'), 'utf8'))
const enTrans = en.translation

// 品牌名、URL、技术术语 —— 跳过这些
const skipPatterns = [
  /^https?:\/\//, /^smtp\./, /^socks5:/, /^name@/, /^noreply@/,
  /^org-/, /^price_/, /^whsec_/, /^edit_this$/, /^my-status$/,
  /^_copy$/, /^gpt-/, /^checkout\./, /^footer\./, /^\[?\{/,
  /^"default/, /^\/status\//, /^\/your\//, /^example\.com/,
  /^AZURE_/, /^AccessKey/, /^OAuth/, /^Client /, /^Webhook URL/,
  /^API URL$/, /^Well-Known/, /^Worker URL$/, /^Uptime Kuma/,
  /^New API/, /^Baidu V2$/, /^Zhipu V4$/, /^Quota:$/,
]

const brandNames = new Set([
  'AIGC2D','Anthropic','API2GPT','Claude','Cloudflare','Cohere','DeepSeek',
  'Discord','DoubaoVideo','FastGPT','Gemini','GitHub','Jimeng','JustSong',
  'LingYiWanWu','LinuxDO','Midjourney','MidjourneyPlus','MiniMax','Mistral',
  'MokaAI','Moonshot','NewAPI','OhMyGPT','Ollama','OpenAI','OpenAIMax',
  'OpenRouter','Passkey','Perplexity','QuantumNous','Replicate','SiliconFlow',
  'Stripe','Submodel','SunoAPI','Telegram','Tencent','Vertex AI','VolcEngine',
  'WeChat','Xinference','Xunfei','AI Proxy','One API',
])

const locales = ['fr', 'ja', 'ru', 'zh', 'zh-TW', 'vi']

for (const locale of locales) {
  const locFile = JSON.parse(await fs.readFile(path.join(LOCALES_DIR, `${locale}.json`), 'utf8'))
  const locTrans = locFile.translation
  const untranslated = {}

  for (const [key, enVal] of Object.entries(enTrans)) {
    const locVal = locTrans[key]
    if (locVal === undefined || locVal !== enVal) continue
    if (brandNames.has(key)) continue
    if (skipPatterns.some(p => p.test(key))) continue
    if (typeof enVal === 'string' && enVal.length < 4) continue
    if (/[a-zA-Z]{3,}/.test(String(enVal))) untranslated[key] = enVal
  }

  const count = Object.keys(untranslated).length
  if (count > 0) {
    console.log(`\n=== ${locale} (${count} untranslated) ===`)
    for (const [k, v] of Object.entries(untranslated))
      console.log(`  ${JSON.stringify(k)}: ${JSON.stringify(v)}`)
  } else {
    console.log(`\n=== ${locale}: all translated ===`)
  }
}
```

### 第 4 步：添加翻译

该脚本是写入语言文件取值的唯一被认可方式。你绝不能通过手工填写 JSON 文件绕过它。
创建 `new-api-web/scripts/add-missing-keys.mjs`，结构如下：

```javascript
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
  en: { /* "key": "English value" */ },
  zh: { /* "key": "中文翻译" */ },
  'zh-TW': { /* "key": "繁體中文翻譯" */ },
  fr: { /* "key": "Traduction française" */ },
  ja: { /* "key": "日本語翻訳" */ },
  ru: { /* "key": "Русский перевод" */ },
  vi: { /* "key": "Bản dịch tiếng Việt" */ },
}

async function main() {
  let totalAdded = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (!Object.prototype.hasOwnProperty.call(json.translation, key)) {
        json.translation[key] = value
        count++
      } else if (json.translation[key] !== value) {
        json.translation[key] = value
        count++
      }
    }

    if (count > 0) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
      )
      await fs.writeFile(filePath, stableStringify(json), 'utf8')
    }

    console.log(`${locale}: ${count} translations applied`)
    totalAdded += count
  }

  console.log(`\nTotal: ${totalAdded} translations applied`)
}

main().catch((err) => { console.error(err); process.exitCode = 1 })
```

为每个语言环境在 `newKeys` 对象中填入实际翻译。

### 第 5 步：验证与清理

```bash
cd new-api-web
node scripts/add-missing-keys.mjs   # 应用翻译
node scripts/find-missing-keys.mjs  # 验证：应提示 "All t() keys found"
bun run i18n:sync                   # 归一化文件顺序
```

完成后删除临时脚本。

## 翻译指南

### 源文本规则

- 翻译每个键之前重新审视它的界面含义：组件位置、用户动作、占位变量、按钮／标签／toast／对话框／校验上下文，
  以及这段文案是名词、命令、状态还是完整句子。
- 优先以英文键或 `en` 取值为源文本。只用调用处来澄清含义、语气与约束。
- 不要把聊天消息、审查评论、issue 描述或任务措辞当作翻译文本。
- 如果源文本含义不清，先检查代码与语言文件。只有当预期的界面文本仍有歧义时，才向用户询问确切文案。

### 长度与布局意识

- 在确定最终措辞前，考虑翻译后的文本是否可能溢出界面，尤其是按钮、表头、菜单项、标签、toast、
  对话框标题、标签页、徽章与空状态。
- 对相对英文常常更长的语言，尤其是法语、俄语与越南语，优先选择自然但紧凑的措辞。
- 不要为了缩短文本而牺牲含义。当调用处空间有限时，选择保留界面意图的最短清晰译法。
- 对带插值的变量、数量、模型名、供应商名、额度与日期，要考虑最长的实际渲染文本，而不只是译文字符串本身。

| 语言 | 代码 | 备注 |
|----------|------|-------|
| 英语 | en | 基准语言，键 = 值 |
| 简体中文 | zh | 回退语言，必须完整 |
| 繁体中文 | zh-TW | 使用自然的繁体中文措辞 |
| 法语 | fr | 很多英语同源词可以直接用（例如 "Configuration"） |
| 日语 | ja | 技术外来词使用片假名 |
| 俄语 | ru | 使用正式语体 |
| 越南语 | vi | 使用标准越南语 |

**保持英文（不翻译）：**
- 品牌／产品名（OpenAI、Claude、Gemini 等）
- URL 与邮箱占位符
- 技术标识符（JSON 键、API 路径、模型名）
- 形似代码的字符串（gpt-3.5-turbo、price_xxx 等）

**始终要翻译：**
- UI 标签、按钮文案、错误信息、描述
- 时间单位（小时、分钟、月、年）
- 动作词（移动、显示、删除等）

## 键的规则

1. 所有脚本都在 `new-api-web/` 目录下运行
2. 使用 `node scripts/xxx.mjs`（ESM 格式，支持顶层 await）
3. 写入语言文件时按字母序排序键
4. 始终以 `bun run i18n:sync` 作为最后一步
5. 完成后删除临时脚本
6. 键中的 `{{variable}}` 占位符必须在所有翻译中保留
7. 绝不要直接编辑 `locales/*.json`。对语言文件的任何非脚本写入（StrReplace、Write、手工改 JSON）都不合规，
   包括只修一个键的情况。
