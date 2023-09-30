package cache

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestCache_Get(t *testing.T) {
	type testCase struct {
		name string
		d    Cache[string]
		want string
	}
	tests := []testCase{
		{
			name: "cache too old",
			d: Cache[string]{
				getNew: func() (string, error) {
					return "new", nil
				},
				currentCache:   "cache",
				maxCacheAge:    time.Second * 5,
				lastUpdateTime: time.Now().Add(time.Second * -10),
				lock:           &sync.Mutex{},
			},
			want: "new",
		},
		{
			name: "use cache",
			d: Cache[string]{
				getNew: func() (string, error) {
					return "new", nil
				},
				currentCache:   "cache",
				maxCacheAge:    time.Second * 5,
				lastUpdateTime: time.Now().Add(time.Second * -2),
				lock:           &sync.Mutex{},
			},
			want: "cache",
		},
		{
			name: "error getting new",
			d: Cache[string]{
				getNew: func() (string, error) {
					return "new", errors.New("test")
				},
				currentCache:   "cache",
				maxCacheAge:    time.Second * 5,
				lastUpdateTime: time.Now().Add(time.Second * -10),
				lock:           &sync.Mutex{},
			},
			want: "cache",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.Get(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkCache_Get(b *testing.B) {
	benchmarks := []struct {
		name           string
		d              Cache[string]
		maxCacheAge    time.Duration
		lastUpdateTime time.Time
	}{
		{
			name: "cached",
			d: Cache[string]{
				getNew: func() (string, error) {
					return "new", errors.New("test")
				},
				currentCache: "cache",
				lock:         &sync.Mutex{},
			},
			maxCacheAge:    time.Second * 5,
			lastUpdateTime: time.Now().Add(time.Second * -2),
		},
		{
			name: "get new",
			d: Cache[string]{
				getNew: func() (string, error) {
					return "new", errors.New("test")
				},
				currentCache: "cache",
				lock:         &sync.Mutex{},
			},
			maxCacheAge:    time.Second * 5,
			lastUpdateTime: time.Now().Add(time.Second * -10),
		},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				bm.d.maxCacheAge = bm.maxCacheAge
				bm.d.lastUpdateTime = bm.lastUpdateTime
				b.StartTimer()

				bm.d.Get()
			}
		})
	}
}
