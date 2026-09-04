[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$failures = [System.Collections.Generic.List[string]]::new()

function Write-CheckSuccess {
    param([string]$Message)
    Write-Host "[OK] $Message" -ForegroundColor Green
}

function Write-CheckFailure {
    param([string]$Message)
    $script:failures.Add($Message)
    Write-Host "[NG] $Message" -ForegroundColor Red
}

function Get-Executable {
    param([string[]]$Names)

    foreach ($name in $Names) {
        $command = Get-Command $name -ErrorAction SilentlyContinue
        if ($null -ne $command) {
            return $command.Source
        }
    }
    return $null
}

function Test-ToolVersion {
    param(
        [string]$Label,
        [string[]]$Commands,
        [string[]]$Arguments,
        [string]$Pattern,
        [version]$MinimumVersion,
        [string]$InstallAdvice
    )

    $executable = Get-Executable $Commands
    if ([string]::IsNullOrWhiteSpace($executable)) {
        Write-CheckFailure "$Label が見つかりません。$InstallAdvice"
        return
    }

    $output = (& $executable @Arguments 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) {
        Write-CheckFailure "$Label の実行に失敗しました。$InstallAdvice"
        return
    }

    $match = [regex]::Match($output, $Pattern)
    if (-not $match.Success) {
        Write-CheckFailure "$Label のバージョンを判定できませんでした。出力: $output"
        return
    }

    $version = [version]$match.Groups[1].Value
    if ($version -lt $MinimumVersion) {
        Write-CheckFailure "$Label $version は要件 $MinimumVersion 以上を満たしません。$InstallAdvice"
        return
    }

    Write-CheckSuccess "$Label $version"
}

Write-Host "FoundryCodex Bridge development environment"

Test-ToolVersion -Label "Go" -Commands @("go.exe", "go") -Arguments @("version") -Pattern 'go(\d+\.\d+(?:\.\d+)?)' -MinimumVersion ([version]"1.25.0") -InstallAdvice "mise run init を実行してください。"
Test-ToolVersion -Label "Node.js" -Commands @("node.exe", "node") -Arguments @("--version") -Pattern 'v?(\d+\.\d+\.\d+)' -MinimumVersion ([version]"18.0.0") -InstallAdvice "Node.js 18以上を https://nodejs.org/ からインストールしてください。"
Test-ToolVersion -Label "npm" -Commands @("npm.cmd", "npm") -Arguments @("--version") -Pattern '(\d+\.\d+\.\d+)' -MinimumVersion ([version]"9.0.0") -InstallAdvice "Node.jsを再インストールしてください。"
Test-ToolVersion -Label "Wails" -Commands @("wails.exe", "wails") -Arguments @("version") -Pattern 'v?(\d+\.\d+\.\d+)' -MinimumVersion ([version]"2.14.0") -InstallAdvice "mise run init を実行してください。"
Test-ToolVersion -Label "Azure CLI" -Commands @("az.cmd", "az") -Arguments @("version", "--output", "json") -Pattern '"azure-cli"\s*:\s*"(\d+\.\d+\.\d+)"' -MinimumVersion ([version]"2.0.81") -InstallAdvice "https://learn.microsoft.com/cli/azure/install-azure-cli-windows を参照してください。"

$wails = Get-Executable @("wails.exe", "wails")
if (-not [string]::IsNullOrWhiteSpace($wails)) {
    $doctorOutput = (& $wails doctor 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -eq 0) {
        Write-CheckSuccess "Wails system dependencies (WebView2を含む)"
    } else {
        Write-CheckFailure "Wails system dependenciesを満たしていません。`n$doctorOutput"
    }
}

if (Test-Path (Join-Path $PSScriptRoot "..\frontend\node_modules")) {
    Write-CheckSuccess "frontend dependencies"
} else {
    Write-CheckFailure "frontend dependenciesがありません。mise run init を実行してください。"
}

if ($failures.Count -gt 0) {
    Write-Host ""
    Write-Host "Doctor failed with $($failures.Count) problem(s)." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Doctor passed." -ForegroundColor Green
