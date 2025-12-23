package cas

import (
	"container/list"
	"io"
)

// LRUCache is a CAS wrapper that caches deserialized objects using LRU eviction
type LRUCache struct {
	underlying CAS
	cache      map[Hash]*list.Element
	evictList  *list.List
	maxSize    int
}

type cacheEntry struct {
	hash  Hash
	value []byte
}

// NewLRUCache creates a new LRU-cached CAS wrapper
// maxSize is the maximum number of entries to cache (0 or negative means unlimited)
func NewLRUCache(underlying CAS, maxSize int) *LRUCache {
	if maxSize <= 0 {
		maxSize = 1000 // Default cache size
	}
	return &LRUCache{
		underlying: underlying,
		cache:      make(map[Hash]*list.Element),
		evictList:  list.New(),
		maxSize:    maxSize,
	}
}

// Put stores an item in the underlying CAS
func (l *LRUCache) Put(item Hashable) (Hash, error) {
	return l.underlying.Put(item)
}

// Has checks if the hash exists in underlying CAS
func (l *LRUCache) Has(hash Hash) bool {
	return l.underlying.Has(hash)
}

// getReader is required by CAS interface but not used directly
func (l *LRUCache) getReader(hash Hash) (bool, io.Reader, error) {
	return l.underlying.getReader(hash)
}

// getValue implements directStore interface - this is where caching happens
func (l *LRUCache) getValue(h Hash) (bool, []byte, error) {
	// Check cache first
	if elem, ok := l.cache[h]; ok {
		// Move to front (most recently used)
		l.evictList.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		return true, entry.value, nil
	}

	// Not in cache - fetch from underlying store
	underlying, ok := l.underlying.(directStore)
	if !ok {
		// Underlying CAS doesn't support direct retrieval
		return false, nil, nil
	}

	has, data, err := underlying.getValue(h)
	if err != nil {
		return false, nil, err
	}
	if !has {
		return false, nil, nil
	}

	// Add to cache
	l.addToCache(h, data)

	return true, data, nil
}

// addToCache adds an entry to the cache and evicts oldest if necessary
func (l *LRUCache) addToCache(hash Hash, value []byte) {
	// If already in cache, update and move to front
	if elem, ok := l.cache[hash]; ok {
		l.evictList.MoveToFront(elem)
		elem.Value.(*cacheEntry).value = value
		return
	}

	// Create new entry
	entry := &cacheEntry{
		hash:  hash,
		value: value,
	}
	elem := l.evictList.PushFront(entry)
	l.cache[hash] = elem

	// Evict oldest if cache is full
	if l.evictList.Len() > l.maxSize {
		l.evictOldest()
	}
}

// evictOldest removes the least recently used entry from cache
func (l *LRUCache) evictOldest() {
	elem := l.evictList.Back()
	if elem != nil {
		l.evictList.Remove(elem)
		entry := elem.Value.(*cacheEntry)
		delete(l.cache, entry.hash)
	}
}

// CacheStats returns cache statistics for monitoring
type CacheStats struct {
	Size    int
	MaxSize int
}

// Stats returns current cache statistics
func (l *LRUCache) Stats() CacheStats {
	return CacheStats{
		Size:    len(l.cache),
		MaxSize: l.maxSize,
	}
}

// RecordWeakStateDepth delegates to underlying CAS
func (l *LRUCache) RecordWeakStateDepth(weakHash Hash, depth int) {
	l.underlying.RecordWeakStateDepth(weakHash, depth)
}

// GetWeakStateDepths delegates to underlying CAS
func (l *LRUCache) GetWeakStateDepths(weakHash Hash) []int {
	return l.underlying.GetWeakStateDepths(weakHash)
}
