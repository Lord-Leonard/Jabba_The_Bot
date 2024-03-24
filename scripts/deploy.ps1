param(
    [Parameter(Mandatory = $true)]
    [string]$RemoteHost,

    [Parameter(Mandatory = $true)]
    [string]$User,

    [string]$DestDir = "/opt/jabba",
    [string]$BinaryName = "bot",
    [string]$Package = "./cmd",
    [string]$GOOS = "linux",
    [string]$GOARCH = "amd64",
    [int]$Port = 22,
    [string]$IdentityFile = "",
    [string]$ServiceName = "",
    [string]$BuildTags = "",
    [string]$BuildArgs = "",
    [switch]$IncludeEnv,
    [switch]$IncludeEmbeddedWeb,
    [switch]$LegacyScp
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if ($ServiceName) {
    $ServiceName = ($ServiceName -replace '\s', '')
}
if ($DestDir) {
    $DestDir = $DestDir.Trim()
}
if ($BinaryName) {
    $BinaryName = $BinaryName.Trim()
}
if ($Package) {
    $Package = $Package.Trim()
}
if ($IdentityFile) {
    $IdentityFile = $IdentityFile.Trim()
    if (-not (Test-Path -LiteralPath $IdentityFile)) {
        throw "Identity file not found: $IdentityFile"
    }
}

function Get-SshArgs {
    $args = @("-p", $Port)
    if ($IdentityFile) {
        $args += @("-i", $IdentityFile)
    }
    $args += "$User@$RemoteHost"
    return $args
}

function Invoke-SshScript {
    param([string]$Script)
    $normalizedScript = (($Script -replace "`r`n", "`n") -replace "`r", "`n").Trim()
    $sshArgs = Get-SshArgs
    # PowerShell writes CRLF to stdin on Windows; strip CR before executing remotely.
    $normalizedScript | & ssh @sshArgs "tr -d '\r' | bash -se"
    $sshExitCode = $LASTEXITCODE
    if ($sshExitCode -ne 0) {
        throw "ssh command failed (exit code: $sshExitCode)"
    }
}

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSCommandPath)
$distDir = Join-Path $repoRoot "dist"
New-Item -ItemType Directory -Force -Path $distDir | Out-Null

$outPath = Join-Path $distDir $BinaryName

Push-Location $repoRoot
try {
    if ($IncludeEmbeddedWeb) {
        $webDir = Join-Path $repoRoot "web"
        $syncScript = Join-Path $repoRoot "scripts/sync_web_embed.ps1"

        if (-not (Test-Path -LiteralPath $webDir)) {
            throw "IncludeEmbeddedWeb set but web directory not found at $webDir"
        }
        if (-not (Test-Path -LiteralPath $syncScript)) {
            throw "IncludeEmbeddedWeb set but sync script not found at $syncScript"
        }

        Write-Host "Building frontend in $webDir"
        Push-Location $webDir
        try {
            & npm run build
            if ($LASTEXITCODE -ne 0) {
                throw "frontend build failed"
            }
        } finally {
            Pop-Location
        }

        Write-Host "Syncing embedded frontend assets"
        & $syncScript
        if ($LASTEXITCODE -ne 0) {
            throw "sync_web_embed.ps1 failed"
        }
    }

    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH
    $env:CGO_ENABLED="0"

    $goArgs = @("build", "-o", $outPath)
    if ($BuildTags) {
        $goArgs += @("-tags", $BuildTags)
    }
    if ($BuildArgs) {
        $goArgs += $BuildArgs.Split(" ")
    }
    $goArgs += $Package

    Write-Host "Building $Package -> $outPath ($GOOS/$GOARCH)"
    & go @goArgs
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed"
    }

    Invoke-SshScript "mkdir -p '$DestDir'"

    $scpArgs = @("-P", $Port)
    if ($LegacyScp) {
        $scpArgs += "-O"
    }
    if ($IdentityFile) {
        $scpArgs += @("-i", $IdentityFile)
    }

    $remoteTmp = "$DestDir/$BinaryName.new"

    if ($ServiceName) {
        Write-Host "Stopping service $ServiceName"
        Invoke-SshScript "sudo systemctl stop '$ServiceName'"
    } else {
        $remoteStop = @'
set -e
PIDFILE="__DEST__/__BIN__.pid"
if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
  kill "$(cat "$PIDFILE")"
  sleep 1
fi
'@
        $remoteStop = $remoteStop.Replace("__DEST__", $DestDir).Replace("__BIN__", $BinaryName)
        Invoke-SshScript $remoteStop
    }

    Write-Host "Uploading binary to ${User}@${RemoteHost}:$remoteTmp"
    & scp @scpArgs $outPath "${User}@${RemoteHost}:$remoteTmp"
    if ($LASTEXITCODE -ne 0) {
        throw "scp binary failed"
    }

    if ($IncludeEnv) {
        $envPath = Join-Path $repoRoot ".env"
        if (-not (Test-Path $envPath)) {
            throw "IncludeEnv set but .env not found at $envPath"
        }
        Write-Host "Uploading .env to ${User}@${RemoteHost}:$DestDir/.env"
        & scp @scpArgs $envPath "${User}@${RemoteHost}:$DestDir/.env"
        if ($LASTEXITCODE -ne 0) {
            throw "scp .env failed"
        }
    }

    Invoke-SshScript "mv -f '$remoteTmp' '$DestDir/$BinaryName'"
    Invoke-SshScript "chmod +x '$DestDir/$BinaryName'"

    if ($ServiceName) {
        Write-Host "Starting service $ServiceName"
        Invoke-SshScript "sudo systemctl start '$ServiceName'"
    } else {
        Write-Host "Starting $BinaryName via nohup"
        $remoteRun = @'
set -e
cd "__DEST__"
PIDFILE="__DEST__/__BIN__.pid"
LOGFILE="__DEST__/__BIN__.log"
if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
  kill "$(cat "$PIDFILE")"
  sleep 1
fi
nohup "__DEST__/__BIN__" >"$LOGFILE" 2>&1 &
echo $! >"$PIDFILE"
'@
        $remoteRun = $remoteRun.Replace("__DEST__", $DestDir).Replace("__BIN__", $BinaryName)
        Invoke-SshScript $remoteRun
    }
} finally {
    Pop-Location
}

Write-Host "Deploy complete."
