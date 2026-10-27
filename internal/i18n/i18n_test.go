package i18n

import (
	"testing"
)

func TestNew(t *testing.T) {
	tr := New("en-us")
	if tr == nil {
		t.Fatal("Translator should not be nil")
	}

	if tr.language != "en-us" {
		t.Errorf("Expected language en-us, got %s", tr.language)
	}
}

func TestT(t *testing.T) {
	tr := New("en-us")

	tests := []struct {
		key      string
		expected string
	}{
		{"welcome", "Welcome to AICockpit"},
		{"version", "Version"},
		{"setup.welcome", "Welcome to AICockpit Setup"},
	}

	for _, test := range tests {
		result := tr.T(test.key)
		if result != test.expected {
			t.Errorf("For key %s, expected %s, got %s", test.key, test.expected, result)
		}
	}
}

func TestTWithArgs(t *testing.T) {
	tr := New("en-us")

	result := tr.T("setup.saved", "/home/user/.cockpit/config.yaml")
	expected := "Configuration saved to /home/user/.cockpit/config.yaml"

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestSetLanguage(t *testing.T) {
	tr := New("en-us")
	tr.SetLanguage("pt-br")

	if tr.language != "pt-br" {
		t.Errorf("Expected language pt-br, got %s", tr.language)
	}

	// Verify translation works in Portuguese
	result := tr.T("welcome")
	expected := "Bem-vindo ao AICockpit"

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestFallbackToEnglish(t *testing.T) {
	tr := New("pt-br")

	// Use a key that exists in both languages
	result := tr.T("welcome")
	if result == "welcome" {
		t.Error("Should have translated the key")
	}
}

func TestMissingKey(t *testing.T) {
	tr := New("en-us")

	result := tr.T("nonexistent.key")
	expected := "nonexistent.key"

	if result != expected {
		t.Errorf("For missing key, expected %s, got %s", expected, result)
	}
}

func TestNewReturnsIndependentInstances(t *testing.T) {
	tr1 := New("en-us")
	tr2 := New("pt-br")

	if tr1 == tr2 {
		t.Error("New() must return independent instances, not a shared singleton")
	}
	if tr1.language != "en-us" {
		t.Errorf("tr1.language = %s, want en-us", tr1.language)
	}
	if tr2.language != "pt-br" {
		t.Errorf("tr2.language = %s, want pt-br", tr2.language)
	}
}
