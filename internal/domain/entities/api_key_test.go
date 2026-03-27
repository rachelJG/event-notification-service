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

func TestClientID(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
		want     string
	}{
		{"with client_id", map[string]string{"client_id": "acme-corp"}, "acme-corp"},
		{"nil metadata", nil, ""},
		{"empty metadata", map[string]string{}, ""},
		{"no client_id key", map[string]string{"org": "acme"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := APIKey{Metadata: tt.metadata}
			if got := key.ClientID(); got != tt.want {
				t.Errorf("ClientID() = %q, want %q", got, tt.want)
			}
		})
	}
}
