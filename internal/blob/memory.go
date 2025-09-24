package blob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

type memBlob struct {
	info Info
	data []byte
}

type memoryStore struct {
	mu   sync.RWMutex
	objs map[string]memBlob
}

func newMemoryStore() *memoryStore { return &memoryStore{objs: make(map[string]memBlob)} }

func (m *memoryStore) Driver() Driver { return DriverMemory }

func (m *memoryStore) Put(ctx context.Context, key string, r io.Reader, opts PutOptions) (Info, error) {
	m.mu.Lock()
	if _, exists := m.objs[key]; exists {
		m.mu.Unlock()
		return Info{}, fmt.Errorf("blob %s already exists", key)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		m.mu.Unlock()
		return Info{}, err
	}
	now := time.Now().UTC()
	info := Info{Key: key, Size: int64(len(b)), ContentType: opts.ContentType, Metadata: cloneMD(opts.Metadata), LastModified: now}
	m.objs[key] = memBlob{info: info, data: b}
	m.mu.Unlock()
	return info, nil
}

func (m *memoryStore) Get(ctx context.Context, key string) (Info, io.ReadCloser, error) {
	m.mu.RLock()
	obj, ok := m.objs[key]
	m.mu.RUnlock()
	if !ok {
		return Info{}, nil, fmt.Errorf("blob %s not found", key)
	}
	dataCopy := make([]byte, len(obj.data))
	copy(dataCopy, obj.data)
	infoCopy := obj.info
	infoCopy.Metadata = cloneMD(infoCopy.Metadata)
	return infoCopy, io.NopCloser(bytes.NewReader(dataCopy)), nil
}

func (m *memoryStore) Head(ctx context.Context, key string) (Info, error) {
	m.mu.RLock()
	obj, ok := m.objs[key]
	m.mu.RUnlock()
	if !ok {
		return Info{}, fmt.Errorf("blob %s not found", key)
	}
	infoCopy := obj.info
	infoCopy.Metadata = cloneMD(infoCopy.Metadata)
	return infoCopy, nil
}

func (m *memoryStore) Delete(ctx context.Context, key string) (bool, error) {
	m.mu.Lock()
	_, ok := m.objs[key]
	if ok {
		delete(m.objs, key)
	}
	m.mu.Unlock()
	return ok, nil
}

func (m *memoryStore) List(ctx context.Context, prefix string) ([]Info, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Info, 0, len(m.objs))
	for k, v := range m.objs {
		if prefix == "" || (len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
			inf := v.info
			inf.Metadata = cloneMD(inf.Metadata)
			out = append(out, inf)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (m *memoryStore) PresignURL(ctx context.Context, key string, opts SignedURLOptions) (string, error) {
	return "", ErrUnsupported
}
