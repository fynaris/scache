package cache

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func TestCacheApi(t *testing.T) {
	cache.Set("1", []byte("Hello World"))
	cache.Set("2", []byte("Hello World 2"), 2*time.Second)
	cache.Set("3", []byte("Hello World 3"))

	if data, ok := cache.Get("1"); !ok || string(data) != "Hello World" {
		t.Errorf("Cannot get the data")
	}

	if _, ok := cache.Get("test"); ok {
		t.Errorf("Get result with no such key.")
	}

	cache.Del("1")
	if _, ok := cache.Get("1"); ok {
		t.Errorf("Get result after delete")
	}

	time.Sleep(2 * time.Second)
	if _, ok := cache.Get("2"); ok {
		t.Errorf("key should be expired")
	}

	cache.Renew("3", 2*time.Second)
	if _, ok := cache.Get("3"); !ok {
		t.Errorf("Should still get the value")
	}

	time.Sleep(2 * time.Second)
	if _, ok := cache.Get("3"); ok {
		t.Errorf("key 3 should be expired")
	}
}

func TestRandomExpire(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	maxExpire := 60
	start := time.Now()
	for i := 1; i < 500000; i++ {
		key := fmt.Sprintf("hello_%d", i)
		expire := rnd.Intn(maxExpire) + 5
		cache.Set(key, []byte("world"), time.Duration(int64(expire))*time.Second)
		if i%2 == 0 {
			if _, ok := cache.Get(key); !ok {
				fmt.Printf("%d: %s, %d - %v, %v\n", i, key, expire, cache.items[key], cache.exps[key])
				t.Errorf("Didn't get the value, %s", key)
			}
		}
	}
	fmt.Printf("%s\n", time.Since(start))
	time.Sleep(90 * time.Second)
}

var cache *Cache

func init() {
	cache = New(5 * time.Second)
	IsDebug = true
	runtime.GOMAXPROCS(runtime.NumCPU())
}
