package version

// Version is the application version.
// Set at build time via -ldflags="-X github.com/lleitep3/aicockpit/internal/version.Version=<ver>"
// Falls back to "dev" when built without ldflags (local builds).
var Version = "dev"

// GetVersion returns the current application version
func GetVersion() string {
	return Version
}
