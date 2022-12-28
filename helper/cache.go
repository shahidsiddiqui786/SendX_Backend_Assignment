package helper

import (
	"errors"
	"fmt"
	"time"
)


/* Initalising the local cache map for the given duration
   Running a clean up loop after each given duration to
   remove the old cache.
*/
func InitLocalCache(cleanupInterval time.Duration) *LocalCache {
	lc := &LocalCache{
		urls: make(map[string]cachedUrl),
		stop:  make(chan struct{}),
	}

	lc.wg.Add(1)
	go func(cleanupInterval time.Duration) {
		defer lc.wg.Done()
		lc.cleanupLoop(cleanupInterval)
	}(cleanupInterval)

	return lc
}

/* Cleanup loop check for entries which
   are expire are removed from cache/map
*/
func (lc *LocalCache) cleanupLoop(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-lc.stop:
			return
		case <-t.C:
			lc.mu.Lock()
			for url, cu := range lc.urls {
				if cu.expireAtTimestamp <= time.Now().Unix() {
					fmt.Println(lc.urls)
					delete(lc.urls, url)
				}
			}
			lc.mu.Unlock()
		}
	}
}

/* Update will update/add the url to the cache
*/
func (lc *LocalCache) Update(u StoreUrl, expireAtTimestamp int64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.urls[u.URL] = cachedUrl{
		StoreUrl: u,
		expireAtTimestamp: expireAtTimestamp,
	}
}

/* updating multiple responses in cache
*/
func UpdateCache(lc *LocalCache,responses []UrlResponse){
	for _, response := range responses {
		lc.Update(StoreUrl{ID: response.ID, URL: response.URI}, time.Now().Add(1*time.Minute).Unix())
	}
}

/* Read would fetch the url in cache 
   returns two value founded url or null
   and error if any
*/
func (lc *LocalCache) Read(url string) (StoreUrl, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	cu, err := lc.urls[url]
	if !err {
		return StoreUrl{}, errors.New("url not found")
	}

	return cu.StoreUrl, nil
}