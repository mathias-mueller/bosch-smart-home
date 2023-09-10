package cache

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Cache[T interface{}] struct {
	getNew         func() (T, error)
	currentCache   T
	maxCacheAge    time.Duration
	lastUpdateTime time.Time
	lock           *sync.Mutex
}

func New[T interface{}](getNew func() (T, error), maxCacheAge time.Duration) *Cache[T] {
	return &Cache[T]{
		lock:           &sync.Mutex{},
		getNew:         getNew,
		maxCacheAge:    maxCacheAge,
		lastUpdateTime: time.Unix(0, 0),
	}
}

func (d *Cache[T]) Get() T {
	d.lock.Lock()
	defer d.lock.Unlock()
	age := time.Since(d.lastUpdateTime)
	if age < d.maxCacheAge {
		log.Debug().
			Time("lastUpdateTime", d.lastUpdateTime).
			Dur("age", age).
			Interface("rooms", d.currentCache).
			Msg("Using cached item")
		return d.currentCache
	}
	log.Debug().
		Dur("age", age).
		Dur("maxAge", d.maxCacheAge).
		Msg("Cached data too old. Refreshing cache...")
	d.lastUpdateTime = time.Now()
	newData, err := d.getNew()
	if err != nil {
		log.Err(err).Msg("Error getting new data")
		return d.currentCache
	}
	d.currentCache = newData
	return d.currentCache
}
