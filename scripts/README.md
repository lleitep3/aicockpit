# AICockpit Installation Scripts

This directory contains platform-specific installation scripts for AICockpit.

## Scripts

### `install.sh` (Linux/macOS)

Intelligent installation script for Unix-like systems.

**Features:**
- Detects installed shells (Bash, Zsh, Fish)
- Automatically adds `~/.local/bin` to PATH
- Checks if PATH is already configured
- Creates shell config files if they don't exist
- Verifies installation after completion
- Provides helpful next steps

**Usage:**
```bash
make install
# or directly
bash scripts/install.sh
```

**What it does:**
1. Creates `~/.local/bin` directory
2. Copies binary to `~/.local/bin/cockpit`
3. Makes binary executable
4. Adds `~/.local/bin` to PATH in:
   - `~/.bashrc` (if Bash is detected)
   - `~/.zshrc` (if Zsh is detected)
   - `~/.config/fish/config.fish` (if Fish is detected)
5. Verifies installation

### `install.ps1` (Windows)

PowerShell installation script for Windows.

**Features:**
- Adds `~/.local/bin` to user PATH (no admin required)
- Checks if PATH is already configured
- Colored output for better readability
- Verifies installation after completion
- Provides helpful next steps

**Usage:**
```powershell
# From PowerShell
make install-win
# or directly
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
```

**What it does:**
1. Creates `~/.local/bin` directory
2. Copies binary to `~/.local/bin/cockpit.exe`
3. Adds `~/.local/bin` to user PATH (permanent)
4. Updates current session PATH
5. Verifies installation

## Installation Paths

| Platform | Install Path | Shell Config |
|----------|--------------|--------------|
| Linux/macOS | `~/.local/bin` | `.bashrc`, `.zshrc`, `config.fish` |
| Windows | `%USERPROFILE%\.local\bin` | User PATH (registry) |

## Features

### Automatic PATH Detection

Both scripts check if `~/.local/bin` is already in PATH before adding it:

```bash
# Linux/macOS
grep -q "\.local/bin" ~/.bashrc

# Windows
$env:PATH -like "*\.local\bin*"
```

### Shell Detection

The Bash script automatically detects which shells you have:

```bash
# Checks for shell config files
[ -f "$HOME/.bashrc" ]    # Bash
[ -f "$HOME/.zshrc" ]     # Zsh
[ -f "$HOME/.config/fish/config.fish" ]  # Fish
```

### Verification

After installation, both scripts verify the binary works:

```bash
# Linux/macOS
command -v cockpit &> /dev/null

# Windows
Get-Command cockpit -ErrorAction SilentlyContinue
```

## Troubleshooting

### PATH not updated

If `~/.local/bin` is not in your PATH after installation:

**Linux/macOS:**
```bash
# Reload shell configuration
source ~/.bashrc  # or ~/.zshrc

# Or open a new terminal
```

**Windows:**
```powershell
# Restart PowerShell or run:
$env:PATH = "$env:USERPROFILE\.local\bin;$env:PATH"
```

### Permission denied

The scripts automatically set execute permissions. If you still get permission errors:

**Linux/macOS:**
```bash
chmod +x ~/.local/bin/cockpit
```

**Windows:**
```powershell
# Usually not needed, but if required:
icacls "$env:USERPROFILE\.local\bin\cockpit.exe" /grant:r "$env:USERNAME`:`(F`)"
```

### Binary not found

Verify the binary was copied:

**Linux/macOS:**
```bash
ls -la ~/.local/bin/cockpit
```

**Windows:**
```powershell
Get-Item "$env:USERPROFILE\.local\bin\cockpit.exe"
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

## Notes

- Scripts are idempotent (safe to run multiple times)
- No sudo/admin required for user-level installation
- Scripts preserve existing PATH entries
- Configuration files are created if they don't exist
- All changes are logged to stdout for transparency
