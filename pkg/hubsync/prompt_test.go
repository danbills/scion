package hubsync

import (
	"testing"
)

func TestConfirmAction_AutoConfirm(t *testing.T) {
	tests := []struct {
		name       string
		defaultYes bool
	}{
		{"defaultYes=true", true},
		{"defaultYes=false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConfirmAction("Test prompt", tt.defaultYes, true)
			if !result {
				t.Errorf("ConfirmAction with autoConfirm=true should always return true, got false (defaultYes=%v)", tt.defaultYes)
			}
		})
	}
}

func TestConfirmAction_NoAutoConfirm_DefaultYes(t *testing.T) {
	// When not auto-confirming and stdin returns EOF/error, it falls back to defaultYes.
	// With defaultYes=true, should return true.
	result := ConfirmAction("Test prompt", true, false)
	if !result {
		t.Error("ConfirmAction with defaultYes=true should return true on stdin EOF")
	}
}

func TestConfirmAction_NoAutoConfirm_DefaultNo(t *testing.T) {
	// When not auto-confirming and stdin returns EOF/error, it falls back to defaultYes.
	// With defaultYes=false, should return false.
	result := ConfirmAction("Test prompt", false, false)
	if result {
		t.Error("ConfirmAction with defaultYes=false should return false on stdin EOF")
	}
}
