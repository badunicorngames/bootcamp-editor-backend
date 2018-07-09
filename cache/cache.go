// Package cache provides resource caching with AppEngine's memcache
package cache

import (
	"errors"

	"appengine"
	"appengine/memcache"
)

type CacheItem interface {
	GetCacheKey() string
	MarshalBinary() (data []byte, err error)
	UnmarshalBinary(data []byte) error
}

var (
	ErrNilCacheItem = errors.New("cache: CacheItem must not be nil")
)

func GetCachedResource(context appengine.Context, cacheItem CacheItem) error {
	if cacheItem == nil {
		return ErrNilCacheItem
	}

	// Check memcache
	item, err := memcache.Get(context, cacheItem.GetCacheKey())
	if err != nil {
		return err
	}

	// Unmarshal and return
	err = cacheItem.UnmarshalBinary(item.Value)
	if err != nil {
		return err
	}

	return nil
}

func CacheResource(context appengine.Context, cacheItem CacheItem) error {
	if cacheItem == nil {
		return ErrNilCacheItem
	}

	// Marshal
	data, err := cacheItem.MarshalBinary()
	if err != nil {
		return err
	}

	// Write to memcache
	item := &memcache.Item{
		Key:   cacheItem.GetCacheKey(),
		Value: data,
	}

	return memcache.Set(context, item)
}

func InvalidateCacheEntry(context appengine.Context, cacheItem CacheItem) error {
	return memcache.Delete(context, cacheItem.GetCacheKey())
}

func InvalidateCacheEntryByKey(context appengine.Context, cacheKey string) error {
	return memcache.Delete(context, cacheKey)
}
