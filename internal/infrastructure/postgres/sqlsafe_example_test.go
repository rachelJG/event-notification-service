package postgres

import "fmt"

func ExampleSafeOrderBy() {
	allowed := map[string]struct{}{
		"created_at": {},
		"type":       {},
	}

	fmt.Println(SafeOrderBy("created_at", allowed, "created_at"))
	fmt.Println(SafeOrderBy("bad;drop table events", allowed, "created_at"))

	// Output:
	// ORDER BY "created_at"
	// ORDER BY "created_at"
}
