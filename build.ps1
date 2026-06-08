#!/usr/bin/env pwsh
# Build script para tlog-pipeline.
# Genera el .syso con metadata de Windows y compila inyectando la version.

param(
    [string]$Version = "5.0.0",
    [string]$Output  = "tlog-gen.exe"
)

$ErrorActionPreference = "Stop"

$root    = $PSScriptRoot
$cmdDir  = Join-Path $root "cmd\pipeline"

# 1) Asegurar que goversioninfo este instalado.
if (-not (Get-Command goversioninfo -ErrorAction SilentlyContinue)) {
    Write-Host "Instalando goversioninfo..."
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
    if ($LASTEXITCODE -ne 0) { throw "fallo go install goversioninfo" }
}

# 2) Generar resource.syso con la metadata de Windows.
Write-Host "Generando resource.syso (metadata Windows)..."
Push-Location $cmdDir
try {
    go generate
    if ($LASTEXITCODE -ne 0) { throw "fallo go generate" }
} finally {
    Pop-Location
}

# 3) Compilar inyectando la version.
Write-Host "Compilando $Output (v$Version)..."
$ldflags = "-s -w -X main.Version=$Version"
go build -ldflags $ldflags -o (Join-Path $root $Output) ./cmd/pipeline
if ($LASTEXITCODE -ne 0) { throw "fallo go build" }

Write-Host "OK: $Output v$Version"
