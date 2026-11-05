---
name: vercel-react-best-practices
description: 来自 Vercel Engineering 的 React 与 Next.js 性能优化指南。在编写、审查或重构涉及组件、Next.js 页面、Server Components、Server Actions、数据获取、包体积、渲染行为或性能改进的 React/Next.js 代码时使用。
---

# Vercel React 最佳实践

在 React 与 Next.js 性能相关工作中使用该技能。完整的 Vercel 指南存放在 `references/full-guide.md`；不要默认整份读取。

## 工作流程

1. 从任务或待审查的代码中确定相关的性能领域。
2. 在 `references/full-guide.md` 中检索匹配的章节或规则标题。
3. 在改动或审查代码之前，只读相关的章节。
4. 优先处理影响更大的类别，其次才是影响较小的微优化。

## 优先级顺序

1. **消除瀑布流**：串行的异步工作、API 路由链、缺失的 `Promise.all`、Suspense 边界。
2. **包体积优化**：桶文件导入（barrel imports）、重量级的客户端模块、动态导入、延迟加载第三方库。
3. **服务端性能**：Server Actions 鉴权、RSC 序列化、按请求去重、跨请求缓存、`after()`。
4. **客户端数据获取**：SWR 去重、全局监听器、被动滚动监听、localStorage 结构。
5. **重渲染优化**：派生状态、effect 依赖、memo 边界、函数式状态更新、transitions、refs。
6. **渲染性能**：hydration 不匹配、长列表、静态 JSX、SVG 精度、资源提示、脚本加载。
7. **JavaScript 性能**：重复查找、数组遍历、存储读取、布局抖动、sort/min-max 的选择。
8. **进阶模式**：一次性初始化、稳定的回调引用、effect events。

## 参考

- 完整的编译版指南：`references/full-guide.md`
- 原始项目：https://github.com/vercel-labs/agent-skills/tree/main/skills/react-best-practices
