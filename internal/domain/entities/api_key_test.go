package entities

import "testing"

func TestHasScope(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		check  string
		want   bool
	}{
		{"scope present", []string{"events:read", "events:write"}, "events:read", true},
		{"scope absent", []string{"events:read"}, "admin", false},
		{"empty scopes", nil, "events:read", false},
		{"exact match required", []string{"events:read"}, "events:rea", false},
		{"single scope match", []string{"admin"}, "admin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := APIKey{Scopes: tt.scopes}
			if got := key.HasScope(tt.check); got != tt.want {
				t.Errorf("HasScope(%q) = %v, want %v", tt.check, got, tt.want)
			}
		})
	}
}
