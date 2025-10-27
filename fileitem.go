package fileitem

import (
	"errors"
	"github.com/k4ties/gq"
	"iter"
	"os"
	"strings"
	"sync"
)

// FileItem is a structure, that allows you to store items either in the text
// file and this structure memory.
type FileItem struct {
	path    string
	items   gq.Set[string]
	itemsMu sync.Mutex
}

// New creates new FileItem instance.
func New(path string) (*FileItem, error) {
	fi := &FileItem{
		path:  path,
		items: make(gq.Set[string]),
	}
	// Initialising the FileItem (load from path)
	if err := fi.load(); err != nil {
		return nil, err
	}
	return fi, nil
}

// Add tries to add item to FileItem memory and sync with file.
func (fi *FileItem) Add(item string) error {
	fi.itemsMu.Lock()
	defer fi.itemsMu.Unlock()

	item = strings.TrimSpace(item)
	if item == "" {
		return errors.New("cannot add empty entry")
	}

	if fi.contains(item) {
		// Already contains this item.
		return errors.New("already exists")
	}

	// Asserting to the memory first
	fi.items.Add(strings.ToLower(item))

	// Then, trying to assert to file
	if err := fi.appendToFile(item); err != nil {
		return err
	}

	return nil
}

// Remove tries to remove the item from FileItem memory.
func (fi *FileItem) Remove(item string) error {
	fi.itemsMu.Lock()
	defer fi.itemsMu.Unlock()

	item = strings.TrimSpace(item)
	if item == "" {
		return errors.New("cannot remove empty entry")
	}

	if fi.contains(item) {
		// Exists in item set, remove it and sync file
		fi.items.Delete(strings.ToLower(item))
		// Sync with the file
		return fi.rewriteFile()
	}

	// Unknown item (not exists in items set)
	return errors.New("entry not found")
}

// Contains returns true, if FileItem has this item in memory. It compares two
// strings by strings.EqualFold method.
func (fi *FileItem) Contains(item string) bool {
	fi.itemsMu.Lock()
	defer fi.itemsMu.Unlock()
	return fi.contains(item)
}

func (fi *FileItem) contains(item string) bool {
	item = strings.TrimSpace(item)
	if item == "" {
		return false
	}
	return fi.items.Contains(strings.ToLower(item))
}

// Items returns the iterator of items.
func (fi *FileItem) Items() iter.Seq[string] {
	fi.itemsMu.Lock()
	defer fi.itemsMu.Unlock()
	return fi.items.Values()
}

// load loads the entries from path.
func (fi *FileItem) load() error {
	data, err := os.ReadFile(fi.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty file, even if it is not exists
			return os.WriteFile(fi.path, []byte{}, 0644)
		}
		// Unexpected error.
		return err
	}

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			// Add, if it is not blank string
			fi.items.Add(strings.ToLower(line))
		}
	}

	return nil
}

// appendToFile appends the item to end of the file.
func (fi *FileItem) appendToFile(item string) error {
	f, err := os.OpenFile(fi.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		// Don't forget to free file
		_ = f.Close()
	}()

	if stat, _ := f.Stat(); stat != nil && stat.Size() > 0 {
		// Adding new line, if file is not empty
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	// Finally, writing the item string to file
	_, err = f.WriteString(item)
	return err
}

// rewriteFile fully rewrites the file with data in memory.
func (fi *FileItem) rewriteFile() error {
	var lines []string
	for item := range fi.items.Values() {
		lines = append(lines, item)
	}
	content := strings.Join(lines, "\n")
	return os.WriteFile(fi.path, []byte(content), 0644)
}
