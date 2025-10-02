// Package memory implements an in-memory blob Store for tests.
package memory

import (
	"bytes"
	"colonycore/internal/blob/core"
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

type blobEntry struct {
	info core.Info
	data []byte
}

// Store implements core.Store backed by process memory. Intended for tests.
type Store struct {
	mu   sync.RWMutex
	objs map[string]blobEntry
}

// New returns an in-memory blob store.
func New() *Store { return &Store{objs: make(map[string]blobEntry)} }

// Driver returns the blob driver identifier.
func (s *Store) Driver() core.Driver { return core.DriverMemory }

// Put stores a new blob; errors if key exists.
func (s *Store) Put(_ context.Context, key string, r io.Reader, opts core.PutOptions) (core.Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.objs[key]; exists {
		return core.Info{}, fmt.Errorf("blob %s already exists", key)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return core.Info{}, err
	}
	now := time.Now().UTC()
	info := core.Info{Key: key, Size: int64(len(b)), ContentType: opts.ContentType, Metadata: cloneMetadata(opts.Metadata), LastModified: now}
	s.objs[key] = blobEntry{info: info, data: b}
	return info, nil
}

// Get returns blob metadata and a read closer to its content.
func (s *Store) Get(_ context.Context, key string) (core.Info, io.ReadCloser, error) {
	s.mu.RLock()
	obj, ok := s.objs[key]
	s.mu.RUnlock()
	if !ok {
		return core.Info{}, nil, fmt.Errorf("blob %s not found", key)
	}
	dataCopy := make([]byte, len(obj.data))
	copy(dataCopy, obj.data)
	infoCopy := obj.info
	infoCopy.Metadata = cloneMetadata(infoCopy.Metadata)
	return infoCopy, io.NopCloser(bytes.NewReader(dataCopy)), nil
}

// Head returns blob metadata only.
func (s *Store) Head(_ context.Context, key string) (core.Info, error) {
	s.mu.RLock()
	obj, ok := s.objs[key]
	s.mu.RUnlock()
	if !ok {
		return core.Info{}, fmt.Errorf("blob %s not found", key)
	}
	infoCopy := obj.info
	infoCopy.Metadata = cloneMetadata(infoCopy.Metadata)
	return infoCopy, nil
}

// Delete removes the blob returning true if it existed.
func (s *Store) Delete(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objs[key]
	if ok {
		delete(s.objs, key)
	}
	return ok, nil
}

// List returns all blobs matching prefix.
func (s *Store) List(_ context.Context, prefix string) ([]core.Info, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]core.Info, 0, len(s.objs))
	for k, v := range s.objs {
		if prefix == "" || (len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
			inf := v.info
			inf.Metadata = cloneMetadata(inf.Metadata)
			out = append(out, inf)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// PresignURL returns unsupported for memory driver.
func (s *Store) PresignURL(_ context.Context, _ string, _ core.SignedURLOptions) (string, error) {
	return "", core.ErrUnsupported
}

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
