package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGeminiSettings(t *testing.T) {
	tmpDir := t.TempDir()
	
	t.Run("ValidJSON", func(t *testing.T) {
		path := filepath.Join(tmpDir, "valid.json")
		content := `{
			"apiKey": "12345",
			"security": { "auth": { "selectedType": "oauth" } },
			"tools": { "sandbox": "container" }
		}`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		s, err := LoadGeminiSettings(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ApiKey != "12345" {
			t.Errorf("expected apiKey 12345, got %q", s.ApiKey)
		}
		if s.Security.Auth.SelectedType != "oauth" {
			t.Errorf("expected auth type oauth, got %q", s.Security.Auth.SelectedType)
		}
		if s.Tools.Sandbox != "container" {
			t.Errorf("expected sandbox container, got %v", s.Tools.Sandbox)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(path, []byte(`{ bad json `), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadGeminiSettings(path)
		if err == nil {
			t.Error("expected error for invalid json, got nil")
		}
	})

	t.Run("MissingFile", func(t *testing.T) {
		_, err := LoadGeminiSettings(filepath.Join(tmpDir, "nonexistent.json"))
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})
}

func TestGetGeminiSettings(t *testing.T) {
	// Mock HOME
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	geminiDir := filepath.Join(tmpHome, ".gemini")
	if err := os.MkdirAll(geminiDir, 0755); err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(geminiDir, "settings.json")
	content := `{"tools": {"sandbox": true}}`
	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s, err := GetGeminiSettings()
	if err != nil {
		t.Fatalf("GetGeminiSettings failed: %v", err)
	}

	if val, ok := s.Tools.Sandbox.(bool); !ok || !val {
		t.Errorf("expected sandbox to be true, got %v", s.Tools.Sandbox)
	}
}
