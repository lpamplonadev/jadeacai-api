package postgres

import "testing"

func TestOpenRequiresDatabaseURL(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}
}
