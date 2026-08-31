package core_test

import "testing"

func requireEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func requireGreaterThan(t *testing.T, got, minimum int) {
	t.Helper()
	if got <= minimum {
		t.Fatalf("want value greater than %d, got %d", minimum, got)
	}
}
