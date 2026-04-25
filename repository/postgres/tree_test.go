package postgres

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestFindPaths_DBError_ReturnsFalse(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}

	// No expectations set → any query returns an error → FindPaths returns false.
	_, _, ok := store.FindPaths("nonexistent_tree", "n1", "n2", 4, 3)
	if ok {
		t.Fatal("expected ok=false on DB error, got true")
	}
}

func TestGetOrCreateTree_DBError_ReturnsError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}

	_, err = store.GetOrCreateTree("ws_1")
	if err == nil {
		t.Fatal("expected error on DB failure, got nil")
	}
}
