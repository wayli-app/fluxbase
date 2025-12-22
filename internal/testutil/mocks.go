// Package testutil provides shared test utilities and mocks for unit testing.
package testutil

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
)

// ErrMockObjectNotFound is returned when an object is not found in mock storage
var ErrMockObjectNotFound = errors.New("object not found")

// MockStorageProvider implements storage.Provider for testing
type MockStorageProvider struct {
	mu      sync.RWMutex
	objects map[string]map[string][]byte // bucket -> key -> data
	buckets map[string]bool

	// Callbacks for custom behavior
	OnUpload   func(ctx context.Context, bucket, key string, data io.Reader, size int64) error
	OnDownload func(ctx context.Context, bucket, key string) (io.ReadCloser, *storage.Object, error)
	OnDelete   func(ctx context.Context, bucket, key string) error
}

// NewMockStorageProvider creates a new mock storage provider
func NewMockStorageProvider() *MockStorageProvider {
	return &MockStorageProvider{
		objects: make(map[string]map[string][]byte),
		buckets: make(map[string]bool),
	}
}

func (m *MockStorageProvider) Name() string {
	return "mock"
}

func (m *MockStorageProvider) Health(ctx context.Context) error {
	return nil
}

func (m *MockStorageProvider) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, opts *storage.UploadOptions) (*storage.Object, error) {
	if m.OnUpload != nil {
		if err := m.OnUpload(ctx, bucket, key, data, size); err != nil {
			return nil, err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.objects[bucket]; !exists {
		m.objects[bucket] = make(map[string][]byte)
	}

	content, _ := io.ReadAll(data)
	m.objects[bucket][key] = content

	return &storage.Object{
		Key:          key,
		Size:         int64(len(content)),
		LastModified: time.Now(),
	}, nil
}

func (m *MockStorageProvider) Download(ctx context.Context, bucket, key string, opts *storage.DownloadOptions) (io.ReadCloser, *storage.Object, error) {
	if m.OnDownload != nil {
		return m.OnDownload(ctx, bucket, key)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if bucketData, exists := m.objects[bucket]; exists {
		if data, exists := bucketData[key]; exists {
			return io.NopCloser(newBytesReader(data)), &storage.Object{Key: key, Size: int64(len(data))}, nil
		}
	}
	return nil, nil, ErrMockObjectNotFound
}

func (m *MockStorageProvider) Delete(ctx context.Context, bucket, key string) error {
	if m.OnDelete != nil {
		return m.OnDelete(ctx, bucket, key)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if bucketData, exists := m.objects[bucket]; exists {
		delete(bucketData, key)
	}
	return nil
}

func (m *MockStorageProvider) Exists(ctx context.Context, bucket, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if bucketData, exists := m.objects[bucket]; exists {
		_, exists := bucketData[key]
		return exists, nil
	}
	return false, nil
}

func (m *MockStorageProvider) GetObject(ctx context.Context, bucket, key string) (*storage.Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if bucketData, exists := m.objects[bucket]; exists {
		if data, exists := bucketData[key]; exists {
			return &storage.Object{Key: key, Size: int64(len(data))}, nil
		}
	}
	return nil, ErrMockObjectNotFound
}

func (m *MockStorageProvider) List(ctx context.Context, bucket string, opts *storage.ListOptions) (*storage.ListResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var objects []storage.Object
	if bucketData, exists := m.objects[bucket]; exists {
		for key, data := range bucketData {
			objects = append(objects, storage.Object{Key: key, Size: int64(len(data))})
		}
	}
	return &storage.ListResult{Objects: objects}, nil
}

func (m *MockStorageProvider) CreateBucket(ctx context.Context, bucket string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buckets[bucket] = true
	m.objects[bucket] = make(map[string][]byte)
	return nil
}

func (m *MockStorageProvider) DeleteBucket(ctx context.Context, bucket string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.buckets, bucket)
	delete(m.objects, bucket)
	return nil
}

func (m *MockStorageProvider) BucketExists(ctx context.Context, bucket string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buckets[bucket], nil
}

func (m *MockStorageProvider) ListBuckets(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var buckets []string
	for bucket := range m.buckets {
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}

func (m *MockStorageProvider) GenerateSignedURL(ctx context.Context, bucket, key string, opts *storage.SignedURLOptions) (string, error) {
	return "https://mock-storage.example.com/" + bucket + "/" + key + "?signed=true", nil
}

func (m *MockStorageProvider) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if srcData, exists := m.objects[srcBucket]; exists {
		if data, exists := srcData[srcKey]; exists {
			if _, exists := m.objects[destBucket]; !exists {
				m.objects[destBucket] = make(map[string][]byte)
			}
			m.objects[destBucket][destKey] = data
			return nil
		}
	}
	return ErrMockObjectNotFound
}

func (m *MockStorageProvider) MoveObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	if err := m.CopyObject(ctx, srcBucket, srcKey, destBucket, destKey); err != nil {
		return err
	}
	return m.Delete(ctx, srcBucket, srcKey)
}

// bytesReader wraps []byte to implement io.Reader
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// MockPubSub implements pubsub.PubSub for testing
type MockPubSub struct {
	mu            sync.RWMutex
	subscriptions map[string][]chan []byte
	published     []PublishedMessage
}

// PublishedMessage records a published message for testing
type PublishedMessage struct {
	Channel string
	Payload []byte
}

// NewMockPubSub creates a new mock pubsub
func NewMockPubSub() *MockPubSub {
	return &MockPubSub{
		subscriptions: make(map[string][]chan []byte),
	}
}

func (m *MockPubSub) Name() string {
	return "mock"
}

func (m *MockPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	m.mu.Lock()
	m.published = append(m.published, PublishedMessage{Channel: channel, Payload: payload})
	subs := m.subscriptions[channel]
	m.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- payload:
		default:
		}
	}
	return nil
}

func (m *MockPubSub) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan []byte, 100)
	m.subscriptions[channel] = append(m.subscriptions[channel], ch)
	return ch, nil
}

func (m *MockPubSub) Unsubscribe(ctx context.Context, channel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subscriptions, channel)
	return nil
}

func (m *MockPubSub) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, subs := range m.subscriptions {
		for _, ch := range subs {
			close(ch)
		}
	}
	m.subscriptions = make(map[string][]chan []byte)
	return nil
}

// GetPublishedMessages returns all published messages for testing
func (m *MockPubSub) GetPublishedMessages() []PublishedMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]PublishedMessage{}, m.published...)
}

// MockSettingsCache provides a mock for auth.SettingsCache
type MockSettingsCache struct {
	mu       sync.RWMutex
	boolVals map[string]bool
	intVals  map[string]int
	strVals  map[string]string
}

// NewMockSettingsCache creates a new mock settings cache
func NewMockSettingsCache() *MockSettingsCache {
	return &MockSettingsCache{
		boolVals: make(map[string]bool),
		intVals:  make(map[string]int),
		strVals:  make(map[string]string),
	}
}

// SetBool sets a boolean value for testing
func (m *MockSettingsCache) SetBool(key string, value bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.boolVals[key] = value
}

// GetBool retrieves a boolean value (mimics SettingsCache.GetBool interface)
func (m *MockSettingsCache) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, exists := m.boolVals[key]; exists {
		return val
	}
	return defaultValue
}

// SetInt sets an integer value for testing
func (m *MockSettingsCache) SetInt(key string, value int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.intVals[key] = value
}

// GetInt retrieves an integer value
func (m *MockSettingsCache) GetInt(ctx context.Context, key string, defaultValue int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, exists := m.intVals[key]; exists {
		return val
	}
	return defaultValue
}

// SetString sets a string value for testing
func (m *MockSettingsCache) SetString(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strVals[key] = value
}

// GetString retrieves a string value
func (m *MockSettingsCache) GetString(ctx context.Context, key string, defaultValue string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, exists := m.strVals[key]; exists {
		return val
	}
	return defaultValue
}
