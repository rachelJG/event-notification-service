package entities

import "time"

// APIKey represents a long-lived authentication key for service consumers.
type APIKey struct {
	ID         string
	KeyHash    string
	Name       string
	Scopes     []string
	IsActive   bool
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// HasScope checks whether this API key has been granted the given scope.
func (k APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
