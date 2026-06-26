package vault

import (
	"testing"

	"github.com/zalando/go-keyring"
)

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
