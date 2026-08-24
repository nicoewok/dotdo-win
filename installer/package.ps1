# PowerShell Build & Packaging Script for dotdo Task Manager
# Compiles binary with embedded resources, creates portable ZIP release, and builds Inno Setup installer.

$ErrorActionPreference = "Stop"

$AppName = "dotdo"
$Version = "1.0.1"
$InstallerDir = $PSScriptRoot
$RootDir = (Get-Item $InstallerDir).Parent.FullName
$DistDir = Join-Path $RootDir "dist"
$BundleDir = Join-Path $DistDir "$AppName-v$Version"
$ZipPath = Join-Path $DistDir "$AppName-v$Version-windows-amd64.zip"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Building & Packaging $AppName v$Version " -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# 1. Clean previous dist directory
if (Test-Path $DistDir) {
    Write-Host "[1/4] Cleaning previous dist directory..." -ForegroundColor Yellow
    Remove-Item -Path $DistDir -Recurse -Force
}
New-Item -ItemType Directory -Path $DistDir | Out-Null
New-Item -ItemType Directory -Path $BundleDir | Out-Null

# 2. Build Windows executable (GUI mode, no command prompt window)
Write-Host "[2/4] Compiling Go executable with embedded assets..." -ForegroundColor Yellow
Push-Location $RootDir
try {
    go build -ldflags="-H=windowsgui" -o dotdo.exe .
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE"
    }
} finally {
    Pop-Location
}
Write-Host "  -> Successfully compiled dotdo.exe" -ForegroundColor Green

# 3. Create Portable Release Bundle & ZIP Archive
Write-Host "[3/4] Creating portable ZIP package..." -ForegroundColor Yellow
Copy-Item (Join-Path $RootDir "dotdo.exe") -Destination $BundleDir
Copy-Item (Join-Path $RootDir "assets") -Destination $BundleDir -Recurse
Copy-Item (Join-Path $RootDir "README.md") -Destination $BundleDir

Compress-Archive -Path "$BundleDir\*" -DestinationPath $ZipPath -Force
Write-Host "  -> Portable ZIP created: $ZipPath" -ForegroundColor Green

# 4. Check for Inno Setup Compiler and build installer
Write-Host "[4/4] Searching for Inno Setup Compiler (ISCC.exe)..." -ForegroundColor Yellow

$isccPath = $null
$possiblePaths = @(
    (Get-Command iscc.exe -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Path),
    "${env:LOCALAPPDATA}\Programs\Inno Setup 6\ISCC.exe",
    "${env:LOCALAPPDATA}\Programs\Inno Setup 5\ISCC.exe",
    "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
    "${env:ProgramFiles}\Inno Setup 6\ISCC.exe",
    "${env:ProgramFiles(x86)}\Inno Setup 5\ISCC.exe",
    "${env:ProgramFiles}\Inno Setup 5\ISCC.exe",
    "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
    "C:\Program Files\Inno Setup 6\ISCC.exe"
)

foreach ($path in $possiblePaths) {
    if ($path -and (Test-Path $path)) {
        $isccPath = $path
        break
    }
}

if ($isccPath) {
    Write-Host "  -> Found ISCC at: $isccPath" -ForegroundColor Green
    Write-Host "  -> Compiling Windows Setup Installer (v$Version)..." -ForegroundColor Yellow
    & $isccPath "/DMyAppVersion=$Version" (Join-Path $InstallerDir "installer.iss")
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  -> Setup Installer created: dist\$AppName-Setup-$Version.exe" -ForegroundColor Green
    } else {
        Write-Warning "ISCC compilation completed with exit code $LASTEXITCODE"
    }
} else {
    Write-Warning "ISCC.exe not found in PATH or standard installation directories."
    Write-Host "  To build setup installer, install Inno Setup 6 (https://jrsoftware.org/isextra.php) and rerun package.ps1, or run iscc installer\installer.iss." -ForegroundColor Cyan
}

Write-Host "`n==========================================" -ForegroundColor Cyan
Write-Host " Packaging Completed Successfully!        " -ForegroundColor Green
Write-Host " Artifacts in: $DistDir" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Get-ChildItem $DistDir | Select-Object Name, Length, LastWriteTime | Format-Table -AutoSize

