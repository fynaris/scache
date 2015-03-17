/*
Package scache provides a simple in-memory cache, supports concurrent access and expirable key/value pairs.

Basic Usage:

	// New a cache with auto clean interval set to 30 seconds
	cache := scache.New(30 * time.Second)

	cache.Set("1", []byte("Hello World"))

	// set key/value and would be expired after 2 seconds
	cache.Set("2", []byte("Hello World 2", 2 * time.Second)

	if data, ok := cache.Get("1"); ok {
		// deal with the data
	}

	cache.Del("1") // will delete the key=1

	// Renew a key to be expired after 15 seconds
	cache.Renew("2", 15 * time.Second)

Expired clear stragety:
1) A key would be cleared when client calls .Get and the key is expired;
2) A goroutine would be running to clear the expired keys and the clearing routine would only be running for less than 50 milliseconds.
*/
package scache

import (
	"log"
	"sync"
	"time"
)

const (
	kDefaultCleanInterval = 30 * time.Second
	kMaxCleanupDuration   = 50 // in milliseconds
)

type Cache struct {
	// will lock the whole object
	sync.RWMutex

	items map[string][]byte
	exps  map[string]int64 // in milliseconds
}

// Save a key/value pair inside cache with or without expiration
func (c *Cache) Set(key string, value []byte, expirations ...time.Duration) {
	expiration := expireSeconds(expirations)
	if expiration == 0 || value == nil {
		c.Del(key)
		return
	}

	c.Lock()
	c.items[key] = value
	if expiration > 0 {
		c.exps[key] = expiration
	}
	c.Unlock()
}

// Get data for a key: data, ok := cache.Get(...), 'ok' would be stand for the key existing.
// If the key is expired at the moment, it would be cleared from the cache.
func (c *Cache) Get(key string) ([]byte, bool) {
	now := time.Now().UnixNano() / 1e6
	var (
		mayExpire bool   = false
		result    []byte = nil
		getit     bool   = false
	)

	c.RLock()
	if exp, ok := c.exps[key]; ok && exp <= now {
		mayExpire = true
	} else {
		result, getit = c.items[key]
	}
	c.RUnlock()

	if mayExpire {
		c.Lock()
		if exp, ok := c.exps[key]; ok && exp <= now {
			delete(c.items, key)
			delete(c.exps, key)
			if IsDebug {
				log.Printf("Lazy clean up for key=%q", key)
			}
		}
		c.Unlock()
	}

	return result, getit
}

// Del key/value from cache
func (c *Cache) Del(key string) {
	c.Lock()
	defer c.Unlock()
	delete(c.items, key)
	delete(c.exps, key)
}

// Renew a key/value for a new expiration time: cache.Renew("test", 0) will delete the key,
// cache.Renew("test") will persist the key/value in the memory
func (c *Cache) Renew(key string, expirations ...time.Duration) {
	expiration := expireSeconds(expirations)
	if expiration == 0 {
		c.Del(key)
		return
	}

	c.Lock()
	defer c.Unlock()
	if _, ok := c.items[key]; ok {
		if expiration > 0 {
			c.exps[key] = expiration
		} else {
			delete(c.exps, key)
		}
	}
}

func expireSeconds(expirations []time.Duration) int64 {
	// all in milliseconds
	var expiration int64 = -1
	if len(expirations) > 0 {
		expiration = int64(expirations[0]) / 1e6
	}
	if expiration == 0 {
		return 0
	} else if expiration > 0 {
		expiration = expiration + time.Now().UnixNano()/1e6
	} else {
		expiration = -1
	}
	return expiration
}

// This will be running inside a goroutine
func (c *Cache) initCleanup(interval time.Duration) {
	cTick := time.Tick(interval)
	for tick := range cTick {
		now := tick.UnixNano() / 1e6
		expireKeys, counter := 0, 0
		c.Lock()
		if len(c.exps) > 0 {
			// since the go map is unordered, the key would be randomized which is great
			for key, expire := range c.exps {
				if expire <= now {
					delete(c.items, key)
					delete(c.exps, key)
					expireKeys++
				}
				counter++
				if counter >= 10000 {
					lasting := time.Now().UnixNano()/1e6 - now
					if lasting > kMaxCleanupDuration {
						break
					}
					counter = 0
				}
			}
			if IsDebug && expireKeys > 0 {
				log.Printf("Cache cleanup, cleaned=%d, size=%d/%d, duration=%s",
					expireKeys, len(c.items), len(c.exps), time.Since(tick))
			}
		}
		c.Unlock()
	}
}

// New a cache object with a user defined clean interval, must be more than 10 seconds.
func New(cleanInterval ...time.Duration) *Cache {
	ci := kDefaultCleanInterval
	if len(cleanInterval) > 0 && cleanInterval[0] >= 10*time.Second {
		ci = cleanInterval[0]
	}

	cache := &Cache{
		items: make(map[string][]byte),
		exps:  make(map[string]int64),
	}
	go func() {
		cache.initCleanup(ci)
	}()
	return cache
}

// Debug switch on the background routine cleanning information, you can ignore this.
var IsDebug bool = false
