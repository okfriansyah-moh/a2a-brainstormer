package executor

import (
	"sync"

	"a2a-brainstorm/agent/internal/config"
	"a2a-brainstorm/agent/internal/llm"
)

type responseCacheEntry struct {
	content      string
	finishReason string
}

// ResponseCache stores LLM responses for exact-hash retries only.
type ResponseCache struct {
	mu    sync.Mutex
	data  map[string]responseCacheEntry
	order []string
	max   int
}

// NewResponseCache creates a retry-only LRU response cache.
func NewResponseCache(max int) *ResponseCache {
	if max < 1 {
		max = 64
	}
	return &ResponseCache{
		data: make(map[string]responseCacheEntry),
		max:  max,
	}
}

var defaultResponseCache = NewResponseCache(config.GetPromptCacheResponseMax())

// Get returns a cached response for hash, if present.
func (c *ResponseCache) Get(hash string) (responseCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[hash]
	if ok {
		c.touch(hash)
	}
	return e, ok
}

// Put stores a response for hash.
func (c *ResponseCache) Put(hash string, content, finishReason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.data[hash]; !exists {
		for len(c.order) >= c.max && len(c.order) > 0 {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.data, oldest)
		}
		c.order = append(c.order, hash)
	} else {
		c.touch(hash)
	}
	c.data[hash] = responseCacheEntry{content: content, finishReason: finishReason}
}

func (c *ResponseCache) touch(hash string) {
	for i, h := range c.order {
		if h == hash {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, hash)
}

// Reset clears the cache (for tests).
func (c *ResponseCache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]responseCacheEntry)
	c.order = nil
}

// lookupRetryCache checks the response cache for an exact request hash.
func lookupRetryCache(req llm.LLMRequest) (generatedContent, bool) {
	if !config.GetPromptCacheEnabled() {
		return generatedContent{}, false
	}
	hash := CanonicalRequestHash(req)
	e, ok := defaultResponseCache.Get(hash)
	if !ok {
		return generatedContent{}, false
	}
	return generatedContent{text: e.content, finishReason: e.finishReason}, true
}

// storeRetryCache saves a successful LLM response for retry reuse.
func storeRetryCache(req llm.LLMRequest, content generatedContent) {
	if !config.GetPromptCacheEnabled() {
		return
	}
	hash := CanonicalRequestHash(req)
	defaultResponseCache.Put(hash, content.text, content.finishReason)
}
