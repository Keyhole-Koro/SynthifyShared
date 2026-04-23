package postgres

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestCreateItem_DBError_ReturnsNil(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}

	// No expectations set → Begin or any query returns an error.
	item := store.CreateItem("tree_1", "New Item", "Description", "", "user_1")
	if item != nil {
		t.Fatal("expected nil on DB error, got item")
	}
}

func TestGetItem_DBError_ReturnsFalse(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}

	_, ok := store.GetItem("nonexistent_item")
	if ok {
		t.Fatal("expected false on DB error, got true")
	}
}
