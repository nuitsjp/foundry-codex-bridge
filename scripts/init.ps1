[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"

function Require-ExternalTool {
    param(
        [string]$Label,
        [string[]]$Commands,
        [string]$InstallAdvice
    )

    foreach ($name in $Commands) {
        if ($null -ne (Get-Command $name -ErrorAction SilentlyContinue)) {
            return
        }
    }

    throw "$Label が見つかりません。$InstallAdvice"
}

function Invoke-NativeCommand {
    param(
        [string]$Command,
        [string[]]$Arguments
    )

    Write-Host "> $Command $($Arguments -join ' ')" -ForegroundColor Cyan
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Command が終了コード $LASTEXITCODE で失敗しました。"
    }
}

function Add-MiseShimsToPath {
    $doctorJson = (& mise doctor --json 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) {
        throw "miseのshimsディレクトリを取得できませんでした。"
    }

    $miseState = $doctorJson | ConvertFrom-Json
    $shimsPath = [string]$miseState.dirs.shims
    if ([string]::IsNullOrWhiteSpace($shimsPath) -or -not (Test-Path $shimsPath)) {
        throw "miseのshimsディレクトリが見つかりません。検出結果: $shimsPath"
    }

    $normalize = {
        param([string]$Path)
        return $Path.Trim().TrimEnd('\', '/').ToLowerInvariant()
    }
    $normalizedShims = & $normalize $shimsPath

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $userEntries = @($userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $userHasShims = $userEntries | Where-Object { (& $normalize $_) -eq $normalizedShims }
    if (-not $userHasShims) {
        $newUserPath = (@($shimsPath) + $userEntries) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        Write-Host "mise shimsをユーザーPATHへ登録しました。現在開いているPowerShellにはターミナル再起動後に反映されます。" -ForegroundColor Yellow
    }

    $processEntries = @($env:Path -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $processHasShims = $processEntries | Where-Object { (& $normalize $_) -eq $normalizedShims }
    if (-not $processHasShims) {
        $env:Path = (@($shimsPath) + $processEntries) -join ';'
    }
}

Require-ExternalTool -Label "Node.js" -Commands @("node.exe", "node") -InstallAdvice "Node.js 18以上を https://nodejs.org/ からインストールしてください。BridgeはNode.jsを自動インストールしません。"
Require-ExternalTool -Label "npm" -Commands @("npm.cmd", "npm") -InstallAdvice "Node.jsを再インストールしてください。"
Require-ExternalTool -Label "Azure CLI" -Commands @("az.cmd", "az") -InstallAdvice "https://learn.microsoft.com/cli/azure/install-azure-cli-windows を参照してください。BridgeはAzure CLIを自動インストールしません。"

$nodeVersionOutput = (& node.exe --version 2>&1 | Out-String).Trim()
$nodeVersionMatch = [regex]::Match($nodeVersionOutput, 'v?(\d+\.\d+\.\d+)')
if (-not $nodeVersionMatch.Success -or [version]$nodeVersionMatch.Groups[1].Value -lt [version]"18.0.0") {
    throw "Node.js 18以上が必要です。検出結果: $nodeVersionOutput"
}

Invoke-NativeCommand -Command "mise" -Arguments @("install", "--jobs", "1")
Add-MiseShimsToPath
Invoke-NativeCommand -Command "mise" -Arguments @("exec", "--", "go", "mod", "download")
Invoke-NativeCommand -Command "npm.cmd" -Arguments @("--prefix", "frontend", "ci")
Invoke-NativeCommand -Command "mise" -Arguments @("run", "doctor")

$miseDoctor = (& mise doctor --json 2>&1 | Out-String).Trim() | ConvertFrom-Json
if (-not $miseDoctor.shims_on_path) {
    throw "mise shimsがPATHに反映されていません。"
}

Write-Host "Initialization completed." -ForegroundColor Green
