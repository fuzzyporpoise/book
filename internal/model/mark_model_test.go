package model

import (
	"errors"
	"path/filepath"
	"testing"

	"go.fuzzyporpoise.dev/book/pkg/book"
)

func testShelfWithMark(t *testing.T, url string) (book.BookShelves, *book.Shelf, *book.Collection) {
	t.Helper()
	shelf, err := book.NewShelf("archive", "")
	if err != nil {
		t.Fatalf("NewShelf: %v", err)
	}
	collection, err := book.NewCollection(shelf, "golang", "")
	if err != nil {
		t.Fatalf("NewCollection: %v", err)
	}
	shelf.AddCollection(collection)

	mark, err := book.NewMarkFromInput(url, nil)
	if err != nil {
		t.Fatalf("NewMarkFromInput: %v", err)
	}
	mark.Shelf = shelf
	mark.Collection = collection
	collection.AddMark(&mark)

	bs := book.BookShelves{*shelf}
	bs.LoadParents()
	return bs, &bs[0], bs[0].Collection("golang")
}

func TestUpdateShelfFileCmdAddRejectsDuplicateURL(t *testing.T) {
	bs, _, collection := testShelfWithMark(t, "https://go.dev")

	dup, err := book.NewMarkFromInput("https://go.dev", nil)
	if err != nil {
		t.Fatalf("NewMarkFromInput: %v", err)
	}
	dup.Shelf = &bs[0]
	dup.Collection = collection

	m := markModel{book: &Book{shelves: &bs}, mark: &dup}
	msg := m.updateShelfFileCmd("add")()

	errMsg, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("updateShelfFileCmd(add) = %T, want errMsg", msg)
	}
	if !errors.Is(errMsg, book.ErrDuplicateURL) {
		t.Errorf("updateShelfFileCmd(add) error = %v, want ErrDuplicateURL", errMsg)
	}
	if got := len(collection.Marks); got != 1 {
		t.Errorf("collection has %d marks, want 1 (duplicate must not be added)", got)
	}
}

func TestUpdateShelfFileCmdAddRejectsTrashedURL(t *testing.T) {
	bs, _, collection := testShelfWithMark(t, "https://go.dev")
	collection.Marks[0].RecordDelete()

	dup, err := book.NewMarkFromInput("https://go.dev", nil)
	if err != nil {
		t.Fatalf("NewMarkFromInput: %v", err)
	}
	dup.Shelf = &bs[0]
	dup.Collection = collection

	m := markModel{book: &Book{shelves: &bs}, mark: &dup}
	msg := m.updateShelfFileCmd("add")()

	errMsg, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("updateShelfFileCmd(add) = %T, want errMsg", msg)
	}
	if !errors.Is(errMsg, book.ErrURLTrashed) {
		t.Errorf("updateShelfFileCmd(add) error = %v, want ErrURLTrashed", errMsg)
	}
}

func TestUpdateShelfFileCmdAddWritesUniqueURL(t *testing.T) {
	bs, shelf, collection := testShelfWithMark(t, "https://go.dev")
	shelf.FilePath = filepath.Join(t.TempDir(), "archive.toml")

	fresh, err := book.NewMarkFromInput("https://example.com", nil)
	if err != nil {
		t.Fatalf("NewMarkFromInput: %v", err)
	}
	fresh.Shelf = shelf
	fresh.Collection = collection

	m := markModel{book: &Book{shelves: &bs}, mark: &fresh}
	msg := m.updateShelfFileCmd("add")()

	if _, ok := msg.(shelfSavedMsg); !ok {
		t.Fatalf("updateShelfFileCmd(add) = %T (%v), want shelfSavedMsg", msg, msg)
	}
	if got := len(collection.Marks); got != 2 {
		t.Errorf("collection has %d marks, want 2", got)
	}
}
