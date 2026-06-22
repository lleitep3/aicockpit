# AICockpit Installation Options

## Overview

AICockpit now supports **two installation modes** to meet different needs:

1. **User-Level Installation** (Recommended for development)
2. **System-Wide Installation** (Recommended for production/distribution)

## Comparison

| Feature | User-Level | System-Wide |
|---------|-----------|------------|
| **Installation Path** | `~/.local/bin` | `/usr/local/bin` |
| **Requires sudo** | ❌ No | ✅ Yes |
| **Available to** | Current user only | All users |
| **Shell config** | Auto-configured | Already in PATH |
| **Uninstall** | Easy (user dir) | Easy (system dir) |
| **Use Case** | Development | Production |

## User-Level Installation

### Command
```bash
make install
```

### What It Does
1. Builds the binary
2. Creates `~/.local/bin` directory
3. Copies binary to `~/.local/bin/cockpit`
4. Detects your shell (Bash, Zsh, Fish)
5. Automatically adds `~/.local/bin` to PATH
6. Verifies installation

### Advantages
- ✅ No sudo required
- ✅ User-specific (doesn't affect other users)
- ✅ Easy to uninstall (just delete `~/.local/bin/cockpit`)
- ✅ Safe to experiment
- ✅ Automatic shell detection

### Disadvantages
- ❌ Only available to current user
- ❌ Requires shell config modification
- ❌ May need shell reload

### When to Use
- Development and testing
- Single-user systems
- Learning and experimentation
- CI/CD pipelines (per-user)

### Example
```bash
$ make install
=== AICockpit User-Level Installation ===

Installing to user location: /home/user/.local/bin

Creating installation directory...
Installing binary...
✓ Binary installed to /home/user/.local/bin/cockpit

Configuring shell...
✓ Bash: PATH already configured
✓ ~/.local/bin is already in your PATH

=== Installation Complete ===
✓ cockpit version 0.1.0
✓ cockpit is ready to use!
```

## System-Wide Installation

### Command
```bash
make install-global
```

### What It Does
1. Builds the binary
2. Creates `/usr/local/bin` directory (with sudo)
3. Copies binary to `/usr/local/bin/cockpit` (with sudo)
4. `/usr/local/bin` is already in system PATH
5. Available to all users immediately
6. Verifies installation

### Advantages
- ✅ Available to all users
- ✅ No shell configuration needed
- ✅ Already in system PATH
- ✅ Standard Linux/Unix practice
- ✅ Perfect for distribution/packaging

### Disadvantages
- ❌ Requires sudo password
- ❌ Affects entire system
- ❌ Requires admin privileges

### When to Use
- Production deployments
- Multi-user systems
- Package distributions
- Docker images
- CI/CD shared runners
- System-wide tools

### Example
```bash
$ make install-global
=== AICockpit Global Installation ===

Installing to system-wide location: /usr/local/bin
This may require sudo

Creating installation directory...
Installing binary...
✓ Binary installed to /usr/local/bin/cockpit

✓ /usr/local/bin is already in system PATH

=== Installation Complete ===
✓ cockpit version 0.1.0
✓ cockpit is ready to use!
```

## Verification

### Check Installation
```bash
# Verify cockpit is accessible
which cockpit

# Check version
cockpit --version

# Test from any directory
cd /tmp && cockpit info
cd ~ && cockpit doctor
cd /var && cockpit setup
```

### Both Installations Are Truly Global
```bash
# User-level installation
$ make install
$ cd /tmp && cockpit --version  # Works!
$ cd /var && cockpit info       # Works!
$ cd /home && cockpit doctor    # Works!

# System-wide installation
$ make install-global
$ cd /tmp && cockpit --version  # Works!
$ cd /var && cockpit info       # Works!
$ cd /home && cockpit doctor    # Works!
```

## Uninstallation

### User-Level
```bash
# Option 1: Using make
make uninstall

# Option 2: Manual
rm ~/.local/bin/cockpit
```

### System-Wide
```bash
# Option 1: Using make (requires sudo)
sudo rm /usr/local/bin/cockpit

# Option 2: Manual
sudo rm /usr/local/bin/cockpit
```

## For Distribution/Packaging

### Recommended Approach
1. **Use system-wide installation** (`make install-global`)
2. **Install to `/usr/local/bin`** (standard location)
3. **Available to all users** immediately
4. **Already in PATH** (no config needed)

### Docker Example
```dockerfile
FROM golang:1.22

WORKDIR /app
COPY . .

# Build and install globally
RUN make install-global

# Now cockpit is available system-wide
RUN cockpit --version
```

### Package Manager Integration
```bash
# In your package build script
make install-global

# cockpit is now available to all users
# and will be included in the package
```

## Shell Configuration

### User-Level Installation
The script automatically configures:

**Bash** (`~/.bashrc`)
```bash
export PATH="$HOME/.local/bin:$PATH"
```

**Zsh** (`~/.zshrc`)
```bash
export PATH="$HOME/.local/bin:$PATH"
```

**Fish** (`~/.config/fish/config.fish`)
```fish
set -gx PATH $HOME/.local/bin $PATH
```

### System-Wide Installation
No configuration needed! `/usr/local/bin` is already in the system PATH:

```bash
$ echo $PATH
/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
```

## Troubleshooting

### User-Level: Command not found
```bash
# Check if ~/.local/bin is in PATH
echo $PATH | grep ".local/bin"

# If not, reload shell
source ~/.bashrc  # or ~/.zshrc

# Or add manually
export PATH="$HOME/.local/bin:$PATH"
```

### System-Wide: Permission denied
```bash
# Make sure you have sudo access
sudo make install-global

# Or manually
sudo cp bin/cockpit /usr/local/bin/
sudo chmod +x /usr/local/bin/cockpit
```

### Both: Binary not found
```bash
# Verify binary exists
ls -la ~/.local/bin/cockpit      # User-level
ls -la /usr/local/bin/cockpit    # System-wide

# Verify it's executable
file ~/.local/bin/cockpit
file /usr/local/bin/cockpit
```

## Recommendations

### For Development
```bash
# Use user-level installation
make install

# Allows experimentation without affecting system
# Easy to uninstall
# No sudo needed
```

### For Production
```bash
# Use system-wide installation
make install-global

# Available to all users
# Standard location
# Easy to distribute
```

### For Distribution/Packaging
```bash
# Use system-wide installation in build scripts
make install-global

# Include in package manager
# Docker images
# CI/CD systems
```

### For AI Systems
```bash
# Either installation works!
# Both are truly global
# Accessible from any directory
# Perfect for AI agent execution
```

## Summary

| Scenario | Recommendation |
|----------|-----------------|
| Personal development | `make install` |
| Team/shared system | `make install-global` |
| Docker/containers | `make install-global` |
| Package distribution | `make install-global` |
| CI/CD pipelines | `make install` (per-user) or `make install-global` |
| AI agent execution | Either (both are global) |

---

**Both installation methods make cockpit truly global and accessible from any directory!** 🚀
