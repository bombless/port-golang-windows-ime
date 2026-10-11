# Build and deploy the standalone candidate-window GUI only. This script does not register the TSF input method.
$ErrorActionPreference = "Stop"

$guiProject = Join-Path $PSScriptRoot "cmd\ime-gui"
$builtGui = Join-Path $guiProject "ime-gui.exe"
$installDir = Join-Path $PSScriptRoot "install"
$installedGui = Join-Path $installDir "ime-gui.exe"

New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Push-Location $guiProject
try {
    & go build -o $builtGui .
    if ($LASTEXITCODE -ne 0) { throw "Candidate GUI build failed with exit code $LASTEXITCODE." }
}
finally { Pop-Location }
if (-not (Test-Path -LiteralPath $builtGui)) { throw "GUI build did not produce $builtGui" }

try {
    Copy-Item -LiteralPath $builtGui -Destination $installedGui -Force -ErrorAction Stop
}
catch {
    throw "Could not deploy ime-gui.exe to '$installedGui'. Close all running ime-gui.exe processes and retry. $($_.Exception.Message)"
}
Write-Host "GUI built and deployed: $installedGui"