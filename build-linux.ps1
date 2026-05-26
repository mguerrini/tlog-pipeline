#!/usr/bin/env pwsh
# Cross-compile para Linux desde Windows.
# No requiere goversioninfo ni resource.syso (son específicos de Windows).
#
# Uso:
#   .\build-linux.ps1                        # versión por defecto, output: tlog-gen
#   .\build-linux.ps1 -Version 2.9.0
#   .\build-linux.ps1 -Version 2.9.0 -Output tlog-gen-linux

param(
    [string]$Version = "2.9.0",
    [string]$Output  = "tlog-gen"
)

$ErrorActionPreference = "Stop"

$root = $PSScriptRoot

Write-Host "Cross-compilando para Linux: $Output v$Version..."

$env:GOOS   = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

try {
    $ldflags = "-s -w -X main.Version=$Version"
    go build -ldflags $ldflags -o (Join-Path $root $Output) ./cmd/pipeline
    if ($LASTEXITCODE -ne 0) { throw "fallo go build" }
} finally {
    # Restaurar variables de entorno para no afectar la sesión
    Remove-Item Env:GOOS
    Remove-Item Env:GOARCH
    Remove-Item Env:CGO_ENABLED
}

Write-Host "OK: $Output v$Version (linux/amd64)"
