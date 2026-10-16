$ErrorActionPreference = "Stop"

$outDir = Join-Path $PSScriptRoot "bin"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$exe = Join-Path $outDir "MediaServiceTray.exe"
go build -trimpath -ldflags "-s -w -H windowsgui" -o $exe .

Write-Host "已生成: $exe"
