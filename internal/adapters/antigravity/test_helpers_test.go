package antigravity

import (
	"database/sql"
	"testing"
	"time"
)

func requireAntigravityEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func requireAntigravityGreater(t *testing.T, got, minimum int) {
	t.Helper()
	if got <= minimum {
		t.Fatalf("want value greater than %d, got %d", minimum, got)
	}
}

func requireAntigravityNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireAntigravityPresent[T any](t *testing.T, value *T) {
	t.Helper()
	if value == nil {
		t.Fatal("expected a value, got nil")
	}
}

func requireAntigravityAbsent[T any](t *testing.T, value *T) {
	t.Helper()
	if value != nil {
		t.Fatal("expected no value, got one")
	}
}

func requireAntigravitySameInstant(t *testing.T, want, got time.Time) {
	t.Helper()
	if !want.Equal(got) {
		t.Fatalf("want %s, got %s", want, got)
	}
}

func requireAntigravityProvenance(t *testing.T, provenance map[string]string, key, want string) {
	t.Helper()
	if got := provenance[key]; got != want {
		t.Fatalf("provenance %q: want %q, got %q", key, want, got)
	}
}

func requireAntigravityExec(t *testing.T, database *sql.DB, query string, args ...interface{}) {
	t.Helper()
	_, err := database.Exec(query, args...)
	if err != nil {
		t.Fatal(err)
	}
}
