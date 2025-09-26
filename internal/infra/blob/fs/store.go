package fs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"colonycore/internal/blob/core"
)

// Store implements core.Store using the local filesystem.
// Keys are mapped to relative file paths under the root. A simple metadata
// sidecar (filename + `.meta`) stores content type & user metadata.
// This is intentionally simple and not concurrent-writer safe beyond per-file creation.
type Store struct {
	root string
}

// New returns a filesystem-backed blob store rooted at path, creating it if needed.
func New(root string) (*Store, error) {
	if root == "" {
		root = "./blobdata"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

func (s *Store) Driver() core.Driver { return core.DriverFilesystem }

// sanitizeKey ensures key doesn't escape root and forbids path traversal and absolute paths.
func sanitizeKey(key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("empty key")
	}
	if strings.Contains(key, "..") {
		return "", fmt.Errorf("invalid key contains '..'")
	}
	if strings.HasPrefix(key, "/") {
		return "", fmt.Errorf("invalid absolute key")
	}
	// normalize separators
	clean := filepath.ToSlash(filepath.Clean(key))
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("invalid key traversal")
	}
	return clean, nil
}

func (s *Store) pathFor(key string) (dataPath, metaPath string, err error) {
	k, err := sanitizeKey(key)
	if err != nil {
		return "", "", err
	}
	dataPath = filepath.Join(s.root, k)
	metaPath = dataPath + ".meta"
	return
}

type metaFile struct {
	ContentType string            `json:"content_type,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ETag        string            `json:"etag"`
	Size        int64             `json:"size"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, opts core.PutOptions) (core.Info, error) {
	dataPath, metaPath, err := s.pathFor(key)
	if err != nil {
		return core.Info{}, err
	}
	// Fail if exists
	if _, err := os.Stat(dataPath); err == nil {
		return core.Info{}, fmt.Errorf("blob %s already exists", key)
	}
	if err := os.MkdirAll(filepath.Dir(dataPath), 0o755); err != nil {
		return core.Info{}, err
	}
	// stream to temp file to compute sha and size
	tmp, err := os.CreateTemp(filepath.Dir(dataPath), ".tmp-*")
	if err != nil {
		return core.Info{}, err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	h := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(tmp, h), r)
	if copyErr != nil {
		_ = tmp.Close()
		return core.Info{}, copyErr
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return core.Info{}, err
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		_ = tmp.Close()
		return core.Info{}, err
	}
	etag := hex.EncodeToString(h.Sum(nil))
	// atomically move into place
	if err := os.Rename(tmp.Name(), dataPath); err != nil {
		_ = tmp.Close()
		return core.Info{}, err
	}
	now := time.Now().UTC()
	mf := metaFile{ContentType: opts.ContentType, Metadata: cloneMetadata(opts.Metadata), ETag: etag, Size: size, CreatedAt: now, UpdatedAt: now}
	if err := writeJSON(metaPath, mf); err != nil {
		return core.Info{}, err
	}
	info := core.Info{Key: key, Size: size, ContentType: opts.ContentType, ETag: etag, Metadata: cloneMetadata(opts.Metadata), LastModified: now, URL: s.localURL(key)}
	return info, nil
}

func (s *Store) Get(ctx context.Context, key string) (core.Info, io.ReadCloser, error) {
	dataPath, metaPath, err := s.pathFor(key)
	if err != nil {
		return core.Info{}, nil, err
	}
	file, err := os.Open(dataPath)
	if errors.Is(err, fs.ErrNotExist) {
		return core.Info{}, nil, err
	}
	if err != nil {
		return core.Info{}, nil, err
	}
	mf, err := readMeta(metaPath)
	if err != nil {
		_ = file.Close()
		return core.Info{}, nil, err
	}
	info := core.Info{Key: key, Size: mf.Size, ContentType: mf.ContentType, ETag: mf.ETag, Metadata: cloneMetadata(mf.Metadata), LastModified: mf.UpdatedAt, URL: s.localURL(key)}
	return info, file, nil
}

func (s *Store) Head(ctx context.Context, key string) (core.Info, error) {
	_, metaPath, err := s.pathFor(key)
	if err != nil {
		return core.Info{}, err
	}
	mf, err := readMeta(metaPath)
	if err != nil {
		return core.Info{}, err
	}
	info := core.Info{Key: key, Size: mf.Size, ContentType: mf.ContentType, ETag: mf.ETag, Metadata: cloneMetadata(mf.Metadata), LastModified: mf.UpdatedAt, URL: s.localURL(key)}
	return info, nil
}

func (s *Store) Delete(ctx context.Context, key string) (bool, error) {
	dataPath, metaPath, err := s.pathFor(key)
	if err != nil {
		return false, err
	}
	_, errData := os.Stat(dataPath)
	if errors.Is(errData, fs.ErrNotExist) {
		return false, nil
	}
	if err := os.Remove(dataPath); err != nil {
		return false, err
	}
	_ = os.Remove(metaPath)
	return true, nil
}

func (s *Store) List(ctx context.Context, prefix string) ([]core.Info, error) {
	// Walk root collecting .meta files and filter prefix.
	var infos []core.Info
	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".meta") {
			mf, err := readMeta(path)
			if err != nil {
				return err
			}
			// derive key
			dataPath := strings.TrimSuffix(path, ".meta")
			rel, err := filepath.Rel(s.root, dataPath)
			if err != nil {
				return err
			}
			key := filepath.ToSlash(rel)
			if prefix == "" || strings.HasPrefix(key, prefix) {
				infos = append(infos, core.Info{Key: key, Size: mf.Size, ContentType: mf.ContentType, ETag: mf.ETag, Metadata: cloneMetadata(mf.Metadata), LastModified: mf.UpdatedAt, URL: s.localURL(key)})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Key < infos[j].Key })
	return infos, nil
}

func (s *Store) PresignURL(ctx context.Context, key string, opts core.SignedURLOptions) (string, error) {
	// Local development convenience: we just return a pseudo URL; no auth.
	if opts.Method != "" && strings.ToUpper(opts.Method) != "GET" {
		return "", core.ErrUnsupported
	}
	return s.localURL(key), nil
}

func (s *Store) localURL(key string) string {
	// Provide a stable opaque URL. Clients can detect dev by scheme host.
	return (&url.URL{Scheme: "http", Host: "local.blob", Path: "/" + key}).String()
}

// --- helpers ---

func cloneMetadata(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func writeJSON(path string, v any) error {
	b, err := jsonMarshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readMeta(path string) (metaFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return metaFile{}, err
	}
	var mf metaFile
	if err := jsonUnmarshal(b, &mf); err != nil {
		return metaFile{}, err
	}
	return mf, nil
}

// isolate json usage to allow later replacement minimal diff.
var (
	jsonMarshal   = func(v any) ([]byte, error) { return json.MarshalIndent(v, "", "  ") }
	jsonUnmarshal = func(b []byte, v any) error { return json.Unmarshal(b, v) }
)
