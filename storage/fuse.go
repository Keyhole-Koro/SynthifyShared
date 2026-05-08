package storage

import (
	"encoding/json"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/synthify/backend/packages/shared/domain"
)

// FUSEHandler manages access to GCS files via a FUSE mount.
type FUSEHandler struct {
	MountPath string
}

// NewFUSEHandler creates a new FUSEHandler with the given mount path.
func NewFUSEHandler(mountPath string) *FUSEHandler {
	return &FUSEHandler{MountPath: mountPath}
}

// ResolvePath returns the absolute local path to a document within the FUSE mount.
// Returns an empty string if MountPath is not set or IDs are missing.
func (h *FUSEHandler) ResolvePath(workspaceID, documentID string) string {
	if h.MountPath == "" || workspaceID == "" || documentID == "" {
		return ""
	}
	return filepath.Join(h.MountPath, workspaceID, documentID)
}

// Open attempts to open a document from the FUSE mount.
// Returns a ReadCloser if the file exists, or nil if FUSE is disabled or the file is missing.
func (h *FUSEHandler) Open(workspaceID, documentID string) (io.ReadCloser, error) {
	path := h.ResolvePath(workspaceID, documentID)
	if path == "" {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, nil
	}

	return os.Open(path)
}

// ReadAll attempts to read the entire content of a document from the FUSE mount.
// Returns the bytes if successful, or nil if FUSE is disabled or the file is missing.
func (h *FUSEHandler) ReadAll(workspaceID, documentID string) ([]byte, error) {
	path := h.ResolvePath(workspaceID, documentID)
	if path == "" {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, nil
	}

	return os.ReadFile(path)
}

// PopulateSourceFile attempts to fill the Content and MimeType of a SourceFile using the FUSE mount.
// Returns true if the file was found and populated, or false if it was not found or FUSE is disabled.
func (h *FUSEHandler) PopulateSourceFile(file *domain.SourceFile) (bool, error) {
	if file == nil {
		return false, nil
	}

	body, err := h.ReadAll(file.WorkspaceID, file.DocumentID)
	if err != nil {
		return false, err
	}
	if body == nil {
		return false, nil
	}

	file.Content = body
	if strings.TrimSpace(file.MimeType) == "" {
		file.MimeType = mime.TypeByExtension(filepath.Ext(file.Filename))
	}
	return true, nil
}

// ResolveCachePath returns the path to a cache entry within the FUSE mount.
func (h *FUSEHandler) ResolveCachePath(documentID, category, key string) string {
	if h.MountPath == "" {
		return ""
	}
	// .cache/v1/{documentID}/{category}/{key}.json
	return filepath.Join(h.MountPath, ".cache", "v1", documentID, category, key+".json")
}

// ReadCache attempts to read and decode a JSON cache entry from the FUSE mount.
// Returns true if found and decoded, or false if missing or FUSE is disabled.
func (h *FUSEHandler) ReadCache(documentID, category, key string, target any) (bool, error) {
	path := h.ResolveCachePath(documentID, category, key)
	if path == "" {
		return false, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(target); err != nil {
		return false, err
	}
	return true, nil
}

// WriteCache encodes and writes data as a JSON cache entry to the FUSE mount.
// Creates parent directories if they don't exist.
func (h *FUSEHandler) WriteCache(documentID, category, key string, data any) error {
	path := h.ResolveCachePath(documentID, category, key)
	if path == "" {
		return nil // FUSE disabled, just skip caching
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Write to .tmp first then rename for atomicity (best effort on GCS FUSE)
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(f).Encode(data); err != nil {
		f.Close()
		return err
	}
	f.Close()

	return os.Rename(tmpPath, path)
}
