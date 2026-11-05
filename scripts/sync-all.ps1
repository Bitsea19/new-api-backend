<#
.SYNOPSIS
    一次同步前后端两个仓库的上游更新（QuantumNous/new-api）。

.DESCRIPTION
    两个仓库与上游的关系不同，因此同步方式也不同：

      new-api-web      是上游 web/ 子目录的独立 fork
                       → 用快照基线三方合并（取上游前端的更新）
                       → scripts/sync-upstream.ps1

      new-api-backend  是整个上游仓库的 fork，但已剥离 web/
                       → 标准三方合并 + 自动剔除上游 web/ 源码
                       → scripts/sync-upstream.ps1

    本脚本按顺序执行两者，并把每一步的结果汇总。

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/sync-all.ps1
    powershell -ExecutionPolicy Bypass -File scripts/sync-all.ps1 -DryRun
#>
param([switch]$DryRun)

$ErrorActionPreference = 'Stop'
# scripts/ 的上一级是仓库根，再上一级是两个仓库的同级目录
$repoRoot   = Split-Path -Parent $PSScriptRoot
$parent     = Split-Path -Parent $repoRoot

$repos = @(
    @{ Name = 'new-api-web';     Path = (Join-Path $parent 'new-api-web') },
    @{ Name = 'new-api-backend'; Path = (Join-Path $parent 'new-api-backend') }
)

$summary = @()

foreach ($repo in $repos) {
    if (-not (Test-Path $repo.Path)) {
        Write-Host "跳过 $($repo.Name)：目录不存在 $($repo.Path)" -ForegroundColor DarkYellow
        continue
    }

    Write-Host ""
    Write-Host "════════ $($repo.Name) ════════" -ForegroundColor Magenta

    Push-Location $repo.Path
    try {
        $script = Join-Path $repo.Path 'scripts/sync-upstream.ps1'
        if (-not (Test-Path $script)) {
            Write-Host "跳过：缺少 scripts/sync-upstream.ps1" -ForegroundColor DarkYellow
            continue
        }

        if ($DryRun) {
            & powershell -ExecutionPolicy Bypass -File $script -DryRun
        } else {
            & powershell -ExecutionPolicy Bypass -File $script
        }
        $exit = $LASTEXITCODE

        $status = if ($exit -eq 0) { 'OK' } else { "失败($exit)" }
        $summary += [pscustomobject]@{ 仓库 = $repo.Name; 结果 = $status }
    }
    finally {
        Pop-Location
    }
}

Write-Host ""
Write-Host "════════ 汇总 ════════" -ForegroundColor Magenta
$summary | Format-Table -AutoSize

if ($DryRun) {
    Write-Host "预演结束，未做任何改动。" -ForegroundColor Yellow
} else {
    Write-Host "请分别检查两个仓库的 git status，确认后提交并推送：" -ForegroundColor Cyan
    Write-Host "  cd new-api-web     && git status && git commit && git push origin master"
    Write-Host "  cd new-api-backend && git status && git commit && git push origin main"
}
