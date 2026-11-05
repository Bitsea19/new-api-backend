---
name: shadcn-ui
description: >-
  为助手提供具备项目上下文的 shadcn/ui 语境：components.json、
  组合模式、CLI、registry、主题定制与 MCP。在处理 Web UI、
  shadcn 组件或预设时使用。概览对齐
  https://ui.shadcn.com/docs/skills.md；完整的上游技能文本已收录在
  vendor/shadcn/ 目录下。
---

<!-- 规范概览：https://ui.shadcn.com/docs/skills.md -->

# Skills（shadcn/ui）

Skills 为 AI 助手提供具备项目上下文的 shadcn/ui 语境。使用后，助手就知道如何用适合本项目的 API 与模式查找、安装、组合和定制组件。

例如，你可以这样问：

- _「加一个带邮箱和密码字段的登录表单。」_
- _「做一个用来更新资料信息的表单设置页。」_
- _「做一个带侧边栏、统计卡片和数据表格的仪表盘。」_
- _「切换到 --preset [CODE]」_
- _「能从 @tailark 加一个 hero 吗？」_

该技能读取你项目的 `components.json`，并提供框架、别名、已安装组件、图标库与基础库，让它一次就能生成正确的代码。

---

## 安装（生态 vs 本仓库）

从 [Skills — shadcn/ui](https://ui.shadcn.com/docs/skills.md) 官方安装：

```bash
npx skills add shadcn/ui
```

那会把技能安装到 `skills` CLI 可用的位置。**本仓库**在 `.agents/skills/shadcn-ui/` 下保留了同样的意图
（这里是概览，**收录**的上游文档在 [`vendor/shadcn/`](./vendor/shadcn/) 中），并从前端应用根目录运行 shadcn CLI：

```bash
cd new-api-web && bunx shadcn@latest info --json
```

在 [skills.sh](https://skills.sh) 上了解更多关于 skills 的内容。

---

## 包含什么（以及在哪里）

### 项目上下文

运行 **`shadcn info --json`**（此处为 `cd new-api-web && bunx shadcn@latest info --json`）
获取框架、Tailwind 版本、别名、基础库（`radix` | `base`）、图标库、已安装组件与解析后的路径。

### CLI 命令

完整命令参考（收录版）：[`vendor/shadcn/cli.md`](./vendor/shadcn/cli.md)。

### 主题与定制

收录版：[`vendor/shadcn/customization.md`](./vendor/shadcn/customization.md)。在线文档：[主题定制](https://ui.shadcn.com/docs/theming)。

### Registry 编写

在 vendor 目录中没有单独复制成一个文件；参见 [Registry](https://ui.shadcn.com/docs/registry) 与 `vendor/shadcn/cli.md` 中的 `build`。

### MCP 服务

收录版：[`vendor/shadcn/mcp.md`](./vendor/shadcn/mcp.md)。在线文档：[MCP Server](https://ui.shadcn.com/docs/mcp)。

---

## 工作方式

1. **项目探测** —— 当 `components.json` 存在时生效（此处为 `new-api-web/components.json`）。
2. **上下文注入** —— 以 `shadcn info --json` 作为 import 与 API 的事实来源。
3. **模式约束** —— 用 [`vendor/shadcn/rules/`](./vendor/shadcn/rules/) 做具体的标记检查；
   更完整的官方工作流参考见下方列表，用于更深入的 CLI、registry 与 preset 问题。
4. **组件发现** —— `shadcn docs`、`shadcn search`、MCP 或 registry —— 需要更深入的上下文时，
   参见官方工作流参考与 MCP 文档。

---

## 了解更多（在线）

- [CLI](https://ui.shadcn.com/docs/cli) —— 与 [`vendor/shadcn/cli.md`](./vendor/shadcn/cli.md) 互补
- [主题定制](https://ui.shadcn.com/docs/theming)
- [Registry](https://ui.shadcn.com/docs/registry)
- [skills.sh](https://skills.sh)

---

## 收录的上游文档包（深度规则）

取自 [shadcn-ui/ui `skills/shadcn`](https://github.com/shadcn-ui/ui/tree/main/skills/shadcn) 的快照；
修订说明见 [`vendor/shadcn/UPSTREAM.txt`](./vendor/shadcn/UPSTREAM.txt)。
上游工作流以参考文件形式保存，并移除了它原有的 skill frontmatter，
这样收录的副本不会被当作第二个本地 skill 发现。

| 文档 | 路径 |
| --- | --- |
| 官方 shadcn/ui 工作流参考 | [`vendor/shadcn/official-shadcn-ui-workflow.md`](./vendor/shadcn/official-shadcn-ui-workflow.md) |
| CLI 参考 | [`vendor/shadcn/cli.md`](./vendor/shadcn/cli.md) |
| 主题／定制 | [`vendor/shadcn/customization.md`](./vendor/shadcn/customization.md) |
| MCP | [`vendor/shadcn/mcp.md`](./vendor/shadcn/mcp.md) |
| 表单 | [`vendor/shadcn/rules/forms.md`](./vendor/shadcn/rules/forms.md) |
| 组合 | [`vendor/shadcn/rules/composition.md`](./vendor/shadcn/rules/composition.md) |
| 图标 | [`vendor/shadcn/rules/icons.md`](./vendor/shadcn/rules/icons.md) |
| 样式 | [`vendor/shadcn/rules/styling.md`](./vendor/shadcn/rules/styling.md) |
| Base 与 Radix | [`vendor/shadcn/rules/base-vs-radix.md`](./vendor/shadcn/rules/base-vs-radix.md) |

**使用方式：** 仓库路径（`new-api-web`、Bun）相关的问题优先看这个**根** `SKILL.md`。
只有在需要完整的官方组件、registry 或 preset 工作流时才读
**`vendor/shadcn/official-shadcn-ui-workflow.md`**。
校验具体标记时使用 **`vendor/shadcn/rules/*.md`**。
