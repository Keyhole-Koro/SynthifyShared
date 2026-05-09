package postgres

import (
	"context"
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
	item := store.CreateItem(context.Background(), "tree_1", "New Item", "Description", "", "user_1")
	if item != nil {
		t.Fatal("expected nil on DB error, got item")
	}
}

func TestGetItem_DBError_ReturnsError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}

	_, err = store.GetItem(context.Background(), "nonexistent_item")
	if err == nil {
		t.Fatal("expected error on DB error, got nil")
	}
}
