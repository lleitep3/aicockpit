# AICockpit Installation Guide

## User-Level Installation (Recommended)

AICockpit installs to `~/.local/bin` by default, which doesn't require sudo.

### Step 1: Build and Install

```bash
cd /path/to/aicockpit
make install
```

This will:
- Build the binary
- Copy it to `~/.local/bin/cockpit`
- Make it executable

### Step 2: Add to PATH

Check if `~/.local/bin` is already in your PATH:

```bash
echo $PATH | grep ".local/bin"
```

If not, add it to your shell configuration:

#### For Bash (~/.bashrc)
```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

#### For Zsh (~/.zshrc)
```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

#### For Fish (~/.config/fish/config.fish)
```bash
echo 'set -gx PATH $HOME/.local/bin $PATH' >> ~/.config/fish/config.fish
source ~/.config/fish/config.fish
```

### Step 3: Verify Installation

```bash
cockpit --version
# Should output: cockpit version 0.1.0
```

## Running Without Installation

If you don't want to install globally, you can run directly from the project:

```bash
cd /path/to/aicockpit
./bin/cockpit setup
./bin/cockpit info
./bin/cockpit doctor
```

## Uninstalling

To remove AICockpit:

```bash
cd /path/to/aicockpit
make uninstall
```

This removes the binary from `~/.local/bin` but keeps your configuration in `~/.cockpit`.

## Troubleshooting

### Command not found after installation

1. Check if `~/.local/bin` is in your PATH:
   ```bash
   echo $PATH
   ```

2. If not, add it:
   ```bash
   export PATH="$HOME/.local/bin:$PATH"
   ```

3. Verify the binary exists:
   ```bash
   ls -la ~/.local/bin/cockpit
   ```

### Permission denied

The `make install` command automatically sets execute permissions. If you get a permission error:

```bash
chmod +x ~/.local/bin/cockpit
```

### Binary not found in PATH

After adding to PATH, you may need to:

1. Open a new terminal window
2. Or reload your shell configuration:
   ```bash
   source ~/.bashrc  # or ~/.zshrc, etc
   ```

## Installation Locations

| Location | Type | Requires sudo | Notes |
|----------|------|---------------|-------|
| `~/.local/bin` | User | No | **Recommended** - No sudo needed |
| `/usr/local/bin` | System | Yes | Requires sudo, affects all users |
| `/bin` | System | Yes | Requires sudo, system directory |
| `./bin/cockpit` | Local | No | Run from project directory |

## Next Steps

After installation, run the setup wizard:

```bash
cockpit setup
```

Then verify everything is working:

```bash
cockpit doctor
```

View your configuration:

```bash
cockpit info
```

---

**Need help?** Check the [QUICK_START.md](QUICK_START.md) or [README.md](README.md) for more information.
