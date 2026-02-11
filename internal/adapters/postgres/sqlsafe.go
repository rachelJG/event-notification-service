package postgres

import "github.com/jackc/pgx/v5"

// SafeIdentifier returns a quoted identifier that is safe to embed in SQL when
// dynamic identifiers are unavoidable. Use an allowlist to restrict values.
func SafeIdentifier(name string, allowed map[string]struct{}, fallback string) string {
	if allowed != nil {
		if _, ok := allowed[name]; !ok {
			name = fallback
		}
	}
	return pgx.Identifier{name}.Sanitize()
}

// SafeOrderBy builds a safe ORDER BY clause using an allowlisted identifier.
func SafeOrderBy(name string, allowed map[string]struct{}, fallback string) string {
	return "ORDER BY " + SafeIdentifier(name, allowed, fallback)
}
