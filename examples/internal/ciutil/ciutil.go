package ciutil

import (
	"fmt"
	"os"
)

// Addr returns the FalkorDB address to use for examples.
// Priority: FALKORDB_TEST_ADDR, then FALKORDB_ADDR, else localhost:6379.
func Addr() string {
	if v := os.Getenv("FALKORDB_TEST_ADDR"); v != "" {
		return v
	}
	if v := os.Getenv("FALKORDB_ADDR"); v != "" {
		return v
	}
	return "localhost:6379"
}

// RequireDB indicates whether examples should fail if FalkorDB is unavailable.
// Set EXAMPLES_REQUIRE_DB=1 in CI to enforce.
func RequireDB() bool {
	return os.Getenv("EXAMPLES_REQUIRE_DB") == "1"
}

func Check(ok bool, format string, args ...any) {
	if ok {
		return
	}
	panic(fmt.Sprintf(format, args...))
}
