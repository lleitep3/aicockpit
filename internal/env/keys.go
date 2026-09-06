// Package env centralizes environment variable names used by the core.
package env

type Key string

const (
	CockpitDataDir  Key = "COCKPIT_DATA_DIR"
	CockpitLogDir   Key = "COCKPIT_LOG_DIR"
	TrackingDir     Key = "TRACKING_DIR"
	CockpitLanguage Key = "COCKPIT_LANGUAGE"
	CockpitVersion  Key = "COCKPIT_VERSION"
	CockpitDevMode  Key = "COCKPIT_DEV_MODE"
	CockpitAppID    Key = "COCKPIT_APP_ID"
)

func (k Key) String() string { return string(k) }
