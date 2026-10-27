package vault

import (
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func newTestPackageVault(t *testing.T, name string) *PackageVault {
	t.Helper()
	keyring.MockInit()
	// Isolate PATH so the "cockpit" CLI cannot be found.
	t.Setenv("PATH", t.TempDir())
	return NewPackageVault(name)
}

func TestPackageVault_Get_Error(t *testing.T) {
	pv := newTestPackageVault(t, "test-package")

	_, err := pv.Get("missing-key")
	if err == nil {
		t.Fatal("expected error when cockpit CLI is unavailable")
	}
	if !strings.Contains(err.Error(), "failed to get secret") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPackageVault_Set_Error(t *testing.T) {
	pv := newTestPackageVault(t, "test-package")

	err := pv.Set("key", "value")
	if err == nil {
		t.Fatal("expected error when cockpit CLI is unavailable")
	}
	if !strings.Contains(err.Error(), "failed to set secret") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPackageVault_Remove_Error(t *testing.T) {
	pv := newTestPackageVault(t, "test-package")

	err := pv.Remove("key")
	if err == nil {
		t.Fatal("expected error when cockpit CLI is unavailable")
	}
	if !strings.Contains(err.Error(), "failed to remove secret") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPackageVault_SetInteractive_Error(t *testing.T) {
	pv := newTestPackageVault(t, "test-package")

	err := pv.SetInteractive("key")
	if err == nil {
		t.Fatal("expected error when cockpit CLI is unavailable")
	}
}

func TestPackageVault_GetWithDefault(t *testing.T) {
	pv := newTestPackageVault(t, "test-package")

	defaultValue := "default"
	value := pv.GetWithDefault("missing-key", defaultValue)
	if value != defaultValue {
		t.Errorf("expected default %q, got %q", defaultValue, value)
	}
}

func TestPackageVault(t *testing.T) {
	// Enable mock keyring for testing
	keyring.MockInit()

	t.Run("Test NewPackageVault", func(t *testing.T) {
		pv := NewPackageVault("test-package")
		if pv == nil {
			t.Fatal("Expected PackageVault, got nil")
		}
		if pv.namespace != "test-package" {
			t.Errorf("Expected namespace 'test-package', got '%s'", pv.namespace)
		}
	})

	t.Run("Test Namespace Sanitization", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"My Package", "my_package"},
			{"my-package", "my-package"},
			{"My/Package", "my_package"},
			{"My Package v1.0", "my_package_v1.0"},
		}

		for _, tc := range testCases {
			pv := NewPackageVault(tc.input)
			if pv.namespace != tc.expected {
				t.Errorf("Input '%s': expected '%s', got '%s'", tc.input, tc.expected, pv.namespace)
			}
		}
	})
}
