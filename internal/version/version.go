package version

// Version is the application version
// This is automatically updated by the release workflow
const Version = "0.1.0"

// GetVersion returns the current application version
func GetVersion() string {
	return Version
}
