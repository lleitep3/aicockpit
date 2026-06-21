package version

import "testing"

func TestGetVersion(t *testing.T) {
	v := GetVersion()
	if v != Version {
		t.Errorf("Expected version %s, got %s", Version, v)
	}
}
