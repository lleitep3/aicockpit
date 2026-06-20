# AICockpit Installation Script for Windows (PowerShell)
# This script handles user-level installation and PATH configuration

param(
    [switch]$Force = $false
)

# Colors for output
$ErrorColor = "Red"
$SuccessColor = "Green"
$WarningColor = "Yellow"
$InfoColor = "Cyan"

function Write-Info {
    param([string]$Message)
    Write-Host $Message -ForegroundColor $InfoColor
}

function Write-Success {
    param([string]$Message)
    Write-Host $Message -ForegroundColor $SuccessColor
}

function Write-Warning {
    param([string]$Message)
    Write-Host $Message -ForegroundColor $WarningColor
}

function Write-Error {
    param([string]$Message)
    Write-Host $Message -ForegroundColor $ErrorColor
}

$BinaryName = "cockpit.exe"
$BinaryPath = "bin\$BinaryName"
$InstallPath = "$env:USERPROFILE\.local\bin"
$CockpitPath = "$InstallPath\$BinaryName"

Write-Info "=== AICockpit Installation for Windows ==="
Write-Host ""

# Check if binary exists
if (-not (Test-Path $BinaryPath)) {
    Write-Error "Error: Binary not found at $BinaryPath"
    Write-Host "Please run 'go build -o bin/cockpit.exe .' first"
    exit 1
}

# Create install directory
Write-Info "Creating installation directory..."
if (-not (Test-Path $InstallPath)) {
    New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
}

# Copy binary
Write-Info "Installing binary..."
Copy-Item -Path $BinaryPath -Destination $CockpitPath -Force
Write-Success "✓ Binary installed to $CockpitPath"
Write-Host ""

# Check if PATH already contains the directory
$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
$PathAlreadyAdded = $CurrentPath -like "*\.local\bin*"

if ($PathAlreadyAdded) {
    Write-Success "✓ ~/.local/bin is already in your PATH"
} else {
    Write-Info "Adding ~/.local/bin to your PATH..."
    
    # Add to user PATH
    $NewPath = "$InstallPath;$CurrentPath"
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    
    # Also add to current session
    $env:PATH = "$InstallPath;$env:PATH"
    
    Write-Success "✓ Added ~/.local/bin to your PATH"
    Write-Warning "⚠ You may need to restart PowerShell for changes to take effect"
}

Write-Host ""
Write-Success "=== Installation Complete ==="
Write-Host ""

# Verify installation
$CockpitCmd = Get-Command cockpit -ErrorAction SilentlyContinue
if ($CockpitCmd) {
    $Version = & $CockpitPath --version
    Write-Success "✓ $Version"
    Write-Success "✓ cockpit is ready to use!"
} else {
    Write-Warning "cockpit not found in PATH"
    Write-Warning "Try restarting PowerShell or running: `$env:PATH = `"$InstallPath;`$env:PATH`""
}

Write-Host ""
Write-Info "Next steps:"
Write-Host "  1. cockpit setup    # Run the setup wizard"
Write-Host "  2. cockpit doctor   # Verify installation"
Write-Host "  3. cockpit info     # View configuration"
Write-Host ""
