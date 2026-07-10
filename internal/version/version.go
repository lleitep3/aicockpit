package version

// Version is the application version
// This is automatically updated by the release workflow
const Version = "0.4.4"

// GetVersion returns the current application version
func GetVersion() string {
	return Version
}
