package cache

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
	sync.RWMutex
	items map[string][]byte
	exps  map[string]int64 // in milliseconds
}

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

func (c *Cache) Del(key string) {
	c.Lock()
	defer c.Unlock()
	delete(c.items, key)
	delete(c.exps, key)
}

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
		}
		if IsDebug {
			log.Printf("Cache cleanup, cleaned=%d, size=%d/%d, duration=%s",
				expireKeys, len(c.items), len(c.exps), time.Since(tick))
		}
		c.Unlock()
	}
}

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

var IsDebug bool = false
