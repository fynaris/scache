package cache

import (
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

var cache *Cache

func init() {
	cache = New(5 * time.Second)
	runtime.GOMAXPROCS(runtime.NumCPU())
}
