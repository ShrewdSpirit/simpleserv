package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/fnv"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/coocood/freecache"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigFastest

type ResponseCache struct {
	c       *freecache.Cache
	ttl     time.Duration
	maxSize int
}

func NewResponseCache(cachettl time.Duration, cachesize, maxcacheitemsize int) (c *ResponseCache, err error) {
	c = &ResponseCache{
		c:       freecache.NewCache(cachesize),
		ttl:     cachettl,
		maxSize: maxcacheitemsize,
	}

	debug.SetGCPercent(10)

	return
}

func (c *ResponseCache) ServeCacheForRequest(w http.ResponseWriter, req *http.Request) bool {
	if req.Method != "GET" &&
		req.Method != "HEAD" {
		return false
	}

	key := CacheItemKey(req)
	value, err := c.c.Get(key)
	if err == freecache.ErrNotFound {
		return false
	}

	item, err := CacheItemUnmarshal(value)
	if err != nil {
		return false
	}

	for k, v := range item.Header {
		w.Header().Set(k, v[0])
	}

	w.Write(item.Body)

	return true
}

var ErrBigBody = errors.New("Response body is too big")

func (c *ResponseCache) SaveCacheForResponse(req *http.Request, resHeader http.Header, body []byte) error {
	if len(body) > c.maxSize {
		return ErrBigBody
	}

	item := NewCacheItem(resHeader, body)
	bytes, err := item.Marshal()
	if err != nil {
		return err
	}

	return c.c.Set(CacheItemKey(req), bytes, int(c.ttl.Seconds()))
}

type CacheItem struct {
	Header http.Header
	Body   []byte
}

func NewCacheItem(header http.Header, body []byte) *CacheItem {
	return &CacheItem{
		Header: header,
		Body:   body,
	}
}

// uint32 size of header
//  uint32:uint32 key,value size
//  buff
// body

func (c *CacheItem) Marshal() (result []byte, err error) {
	buf := bytes.Buffer{}

	{
		headerSize := make([]byte, 4)
		binary.LittleEndian.PutUint32(headerSize, uint32(len(c.Header)))
		buf.Write(headerSize)
	}

	{
		for k, v := range c.Header {
			keySize := make([]byte, 4)
			valSize := make([]byte, 4)
			binary.LittleEndian.PutUint32(keySize, uint32(len(k)))
			binary.LittleEndian.PutUint32(valSize, uint32(len(v[0])))
			buf.Write(keySize)
			buf.Write(valSize)
			buf.Write([]byte(k))
			buf.Write([]byte(v[0]))
		}
	}

	buf.Write(c.Body)

	result = buf.Bytes()

	return
}

func CacheItemUnmarshal(value []byte) (c *CacheItem, err error) {
	c = &CacheItem{
		Header: make(http.Header),
	}

	r := bytes.NewReader(value)

	headerSizeBytes := make([]byte, 4)
	if _, err = r.Read(headerSizeBytes); err != nil {
		return
	}

	for i := 0; i < int(binary.LittleEndian.Uint32(headerSizeBytes)); i++ {
		keySizeBytes := make([]byte, 4)
		valSizeBytes := make([]byte, 4)
		r.Read(keySizeBytes)
		r.Read(valSizeBytes)

		keyBytes := make([]byte, binary.LittleEndian.Uint32(keySizeBytes))
		valBytes := make([]byte, binary.LittleEndian.Uint32(valSizeBytes))

		r.Read(keyBytes)
		r.Read(valBytes)

		c.Header.Set(string(keyBytes), string(valBytes))
	}

	c.Body = make([]byte, r.Len())
	r.Read(c.Body)

	return
}

func CacheItemKey(req *http.Request) []byte {
	url := strings.TrimSuffix(req.URL.Path, "/")

	hash := fnv.New128a()
	hash.Write([]byte(req.Method))
	hash.Write([]byte(url))

	return hash.Sum(nil)
}
