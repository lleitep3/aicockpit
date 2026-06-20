# AICockpit Installation Scripts - Complete Guide

## Overview

AICockpit now includes intelligent installation scripts that automatically configure your system for both Linux/macOS and Windows.

## What the Scripts Do

### Automatic Features

✅ **Detect your shell** (Bash, Zsh, Fish)  
✅ **Check if PATH already configured** (avoid duplicates)  
✅ **Create shell config files** if they don't exist  
✅ **Add to PATH automatically** (no manual editing needed)  
✅ **Verify installation** after completion  
✅ **Provide next steps** with helpful guidance  

### No Sudo Required

- Installation is user-level (`~/.local/bin`)
- No system-wide changes
- Safe to run multiple times
- Easy to uninstall

## Installation

### Linux/macOS - User-Level (Recommended)

```bash
cd /path/to/aicockpit
make install
```

Installs to `~/.local/bin` (user-specific, no sudo required)

### Linux/macOS - System-Wide (Global)

```bash
cd /path/to/aicockpit
make install-global
```

Installs to `/usr/local/bin` (available to all users, requires sudo)

**User-Level Installation - What happens:**
1. Builds the binary
2. Creates `~/.local/bin` directory
3. Copies binary to `~/.local/bin/cockpit`
4. Detects your shell (Bash, Zsh, Fish)
5. Adds `~/.local/bin` to PATH in:
   - `~/.bashrc` (if Bash is detected)
   - `~/.zshrc` (if Zsh is detected)
   - `~/.config/fish/config.fish` (if Fish is detected)
6. Verifies installation
7. Shows next steps

**System-Wide Installation - What happens:**
1. Builds the binary
2. Creates `/usr/local/bin` directory (with sudo)
3. Copies binary to `/usr/local/bin/cockpit` (with sudo)
4. `/usr/local/bin` is already in system PATH
5. Available to all users on the system
6. Verifies installation
7. Shows next steps

**Example output:**
```
=== AICockpit Installation ===

Creating installation directory...
Installing binary...
✓ Binary installed to /home/user/.local/bin/cockpit

Configuring shell...

✓ Bash: PATH already configured
✓ Zsh: PATH already configured
✓ ~/.local/bin is already in your PATH

=== Installation Complete ===

✓ cockpit version 0.1.0
✓ cockpit is ready to use!

Next steps:
  1. cockpit setup    # Run the setup wizard
  2. cockpit doctor   # Verify installation
  3. cockpit info     # View configuration
```

### Windows

```powershell
cd C:\path\to\aicockpit
make install-win
```

Or directly with PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
```

**What happens:**
1. Builds the binary
2. Creates `~\.local\bin` directory
3. Copies binary to `~\.local\bin\cockpit.exe`
4. Adds `~\.local\bin` to user PATH (permanent)
5. Updates current session PATH
6. Verifies installation
7. Shows next steps

**Example output:**
```
=== AICockpit Installation for Windows ===

Creating installation directory...
Installing binary...
✓ Binary installed to C:\Users\username\.local\bin\cockpit.exe

✓ ~/.local/bin is already in your PATH

=== Installation Complete ===

✓ cockpit version 0.1.0
✓ cockpit is ready to use!

Next steps:
  1. cockpit setup    # Run the setup wizard
  2. cockpit doctor   # Verify installation
  3. cockpit info     # View configuration
```

## Script Details

### `scripts/install.sh` (Linux/macOS)

**Language:** Bash  
**Size:** ~200 lines  
**Dependencies:** Standard Unix tools (grep, mkdir, cp, chmod)

**Features:**
- ANSI color output for better readability
- Detects multiple shells
- Idempotent (safe to run multiple times)
- Checks if PATH already configured
- Creates files if they don't exist
- Comprehensive error handling

**Key functions:**
- `path_already_added()` - Checks if PATH is configured
- `add_to_path()` - Adds to shell config files

### `scripts/install.ps1` (Windows)

**Language:** PowerShell  
**Size:** ~100 lines  
**Requirements:** PowerShell 3.0+

**Features:**
- Colored output for better readability
- Adds to user PATH (registry)
- Updates current session
- Idempotent (safe to run multiple times)
- Comprehensive error handling

**Key functions:**
- `Write-Info()` - Cyan colored output
- `Write-Success()` - Green colored output
- `Write-Warning()` - Yellow colored output
- `Write-Error()` - Red colored output

## How It Works

### Shell Detection (Linux/macOS)

The script checks for shell config files:

```bash
# Bash
[ -f "$HOME/.bashrc" ]

# Zsh
[ -f "$HOME/.zshrc" ]

# Fish
[ -f "$HOME/.config/fish/config.fish" ]
```

### PATH Detection (Linux/macOS)

Before adding to PATH, it checks if already configured:

```bash
grep -q "\.local/bin" "$shell_config"
```

### PATH Configuration (Linux/macOS)

Adds the following line to shell config:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### PATH Configuration (Windows)

Adds `~\.local\bin` to user PATH via:

```powershell
[Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
```

## Verification

After installation, the scripts verify the binary works:

### Linux/macOS

```bash
if command -v cockpit &> /dev/null; then
    VERSION=$($COCKPIT_PATH --version)
    echo "✓ $VERSION"
fi
```

### Windows

```powershell
$CockpitCmd = Get-Command cockpit -ErrorAction SilentlyContinue
if ($CockpitCmd) {
    $Version = & $CockpitPath --version
}
```

## Troubleshooting

### PATH not updated after installation

**Linux/macOS:**
```bash
# Reload shell configuration
source ~/.bashrc  # or ~/.zshrc

# Or open a new terminal window
```

**Windows:**
```powershell
# Restart PowerShell or run:
$env:PATH = "$env:USERPROFILE\.local\bin;$env:PATH"
```

### Binary not found

**Linux/macOS:**
```bash
# Check if binary exists
ls -la ~/.local/bin/cockpit

# Check if it's executable
file ~/.local/bin/cockpit

# Make it executable if needed
chmod +x ~/.local/bin/cockpit
```

**Windows:**
```powershell
# Check if binary exists
Get-Item "$env:USERPROFILE\.local\bin\cockpit.exe"

# Check PATH
$env:PATH -split ";"
```

### Permission denied

**Linux/macOS:**
```bash
chmod +x ~/.local/bin/cockpit
```

**Windows:**
Usually not needed, but if required:
```powershell
icacls "$env:USERPROFILE\.local\bin\cockpit.exe" /grant:r "$env:USERNAME`:`(F`)"
```

## Uninstallation

To remove AICockpit:

```bash
make uninstall
```

This removes the binary but keeps your configuration in `~/.cockpit`.

## Advanced Usage

### Manual Installation

If you prefer to install manually:

**Linux/macOS:**
```bash
mkdir -p ~/.local/bin
cp bin/cockpit ~/.local/bin/
chmod +x ~/.local/bin/cockpit
export PATH="$HOME/.local/bin:$PATH"
```

**Windows:**
```powershell
New-Item -ItemType Directory -Path "$env:USERPROFILE\.local\bin" -Force
Copy-Item -Path "bin\cockpit.exe" -Destination "$env:USERPROFILE\.local\bin\"
[Environment]::SetEnvironmentVariable("PATH", "$env:USERPROFILE\.local\bin;$env:PATH", "User")
```

### Running Multiple Times

The scripts are idempotent and safe to run multiple times:

```bash
# Safe to run again
make install

# Won't add duplicate PATH entries
# Won't overwrite existing config
```

## Development

### Modifying Scripts

When modifying installation scripts:

1. Test on the target platform
2. Verify PATH detection works
3. Test with multiple shells (if applicable)
4. Ensure error handling is robust
5. Update documentation

### Testing

```bash
# Test on Linux/macOS
bash scripts/install.sh

# Test on Windows
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
```

## Files

- `scripts/install.sh` - Linux/macOS installation script
- `scripts/install.ps1` - Windows installation script
- `scripts/README.md` - Script documentation
- `Makefile` - Build automation (calls scripts)

## Summary

The installation scripts provide a seamless, platform-aware installation experience:

✅ **Automatic** - No manual PATH editing  
✅ **Smart** - Detects shells and existing configuration  
✅ **Safe** - No sudo required, idempotent  
✅ **Cross-platform** - Works on Linux, macOS, and Windows  
✅ **User-friendly** - Clear output and helpful guidance  

---

For more information, see:
- [INSTALLATION.md](INSTALLATION.md) - Installation guide
- [QUICK_START.md](QUICK_START.md) - Quick start guide
- [scripts/README.md](scripts/README.md) - Script documentation
