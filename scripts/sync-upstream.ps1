<#
.SYNOPSIS
    从上游 QuantumNous/new-api 同步后端改动到本仓库（二开版），不污染前端目录。

.DESCRIPTION
    背景：
    本仓库已把前端面板剥离到独立仓库 new-api-web，后端仓库不再跟踪 web/ 源码
    （只保留 web/.gitignore 作为「前端产物落点」的占位说明）。
    但上游仓库的 main 分支仍然带着完整的 web/ 目录。

    直接 `git merge upstream/main` 的实测结果：
      - 后端代码（model/、relay/、middleware/ 等）**零冲突**，正常合入
      - 冲突 100% 集中在 web/ 目录，形式为 modify/delete
        （上游修改了 web/ 里的文件，而本仓库早已删除整个 web/ 源码）

    因此本脚本的做法：
      1. git fetch upstream
      2. git merge upstream/main（保留标准的三方合并语义）
      3. 自动把 web/ 目录恢复成「只有 web/.gitignore」的二开形态
      4. 检查剩余冲突（若有，非 web 目录的才是真正需要人工处理的）

    与前端仓库 scripts/sync-upstream.ps1 的分工：
      - 前端仓库：上游的 web/ 子目录 → 快照基线 → 三方合并（取上游 web/ 的更新）
      - 后端仓库：上游整棵树 → 标准 merge → 剔除 web/（不要上游的前端）

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/sync-upstream.ps1
    powershell -ExecutionPolicy Bypass -File scripts/sync-upstream.ps1 -DryRun
#>
param(
    [string]$Upstream = 'upstream',
    [string]$Branch   = 'main',
    [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

function Write-Step { param([string]$Msg) Write-Host "==> $Msg" -ForegroundColor Cyan }

Write-Step "拉取上游 $Upstream/$Branch"
git fetch $Upstream $Branch
if ($LASTEXITCODE -ne 0) { throw "拉取上游失败" }

$upstreamSha = (git rev-parse "$Upstream/$Branch").Trim()
$behind = [int](git rev-list --count "HEAD..$Upstream/$Branch")
Write-Step "上游 commit: $upstreamSha （本仓库落后 $behind 个提交）"

if ($behind -eq 0) {
    Write-Host "已是最新，无需同步。" -ForegroundColor Green
    exit 0
}

# 预演：先看看会冲突什么
if ($DryRun) {
    Write-Step "DryRun：试合并（不提交）"
    git merge --no-commit --no-ff $Upstream/$Branch
    $mergeExit = $LASTEXITCODE

    $allConflicts = @(git diff --name-only --diff-filter=U)
    $webConflicts  = @($allConflicts | Where-Object { $_ -like 'web/*' })
    $realConflicts = @($allConflicts | Where-Object { $_ -notlike 'web/*' })

    Write-Step "冲突总数: $($allConflicts.Count)"
    Write-Step "其中 web/ 目录（将被自动剔除）: $($webConflicts.Count)"
    Write-Step "真正需要人工处理的: $($realConflicts.Count)"
    if ($realConflicts.Count -gt 0) {
        $realConflicts | ForEach-Object { Write-Host "  $_" -ForegroundColor Yellow }
    }

    git merge --abort 2>$null
    Write-Host "DryRun 结束，已回退。" -ForegroundColor Yellow
    exit 0
}

# --- 正式合并 ---
Write-Step "合并上游 $Branch"
git merge --no-commit --no-ff $Upstream/$Branch
$mergeExit = $LASTEXITCODE

# --- 剔除上游带来的 web/ 源码，恢复二开形态 ---
Write-Step "剔除上游的 web/ 源码（前端在独立仓库维护）"
$webFiles = @(git ls-files -- "web/*" | Where-Object { $_ -ne 'web/.gitignore' })
if ($webFiles.Count -gt 0) {
    git rm -r --cached --quiet --ignore-unmatch -- $webFiles
    Write-Host "已从索引移除 $($webFiles.Count) 个 web/ 文件"
}
# 清理工作区残留
git checkout --quiet -- . 2>$null
Get-ChildItem -Path 'web' -Recurse -File -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -notlike '*\.gitignore' } |
    ForEach-Object { Remove-Item $_.FullName -Force -ErrorAction SilentlyContinue }

# 恢复 web/.gitignore（合并可能删掉了它）
if (-not (Test-Path 'web/.gitignore')) {
    New-Item -ItemType Directory -Path 'web' -Force | Out-Null
    git checkout upstream/main -- web/.gitignore 2>$null
    git checkout HEAD -- web/.gitignore 2>$null
}

# --- 报告真正的冲突 ---
$allConflicts = @(git diff --name-only --diff-filter=U)
$realConflicts = @($allConflicts | Where-Object { $_ -notlike 'web/*' })

Write-Step "冲突总数: $($allConflicts.Count)（已自动清理 web/ 部分）"
if ($realConflicts.Count -gt 0) {
    Write-Host "以下文件需要人工处理（上游与二开都改过）：" -ForegroundColor Yellow
    $realConflicts | ForEach-Object { Write-Host "  $_" }
    Write-Host "保留二开版本: git checkout --ours <文件>" -ForegroundColor DarkGray
    Write-Host "采用上游版本: git checkout --theirs <文件>" -ForegroundColor DarkGray
    Write-Host "处理后执行: git add <文件> && git commit" -ForegroundColor DarkGray
} else {
    Write-Host "无冲突，上游后端改动已自动合入，web/ 已保持二开形态。" -ForegroundColor Green
}

Write-Step "变更清单"
git status --short
Write-Step "完成：检查后执行 git commit 提交合并结果"
