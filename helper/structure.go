package helper

import (
	"sync"
)

type StoreUrl struct {
	URL string `json:"url"`
	ID string `json:"id"`
}

type cachedUrl struct {
	StoreUrl
	expireAtTimestamp int64
}

type LocalCache struct {
	stop chan struct{}

	wg    sync.WaitGroup
	mu    sync.RWMutex
	urls map[string]cachedUrl
}

type UrlRequest struct {
	URI        string `json:"uri"`
	RetryLimit int    `json:"retryLimit"`
}

type UrlResponse struct {
	ID        string `json:"id"`
	URI       string `json:"uri"`
	SourceURI string `json:"sourceUri"`
}

type UrlMultiRequest struct {
	URLS []UrlRequest `json:"urls"`
}

type UrlMultiResponse struct {
	URLS    []UrlResponse `json:"urls"`
	Failed  []UrlRequest  `json:"failed"`
}