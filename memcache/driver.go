package memcache

import (
	"errors"
	"fmt"
	"log"
	"time"

	youtubeMemcache "github.com/youtube/vitess/go/memcache"
)

// Memcache struct
type Memcache struct {
	conn *youtubeMemcache.Connection
}

// Kv : json
type Kv struct {
	// Key name
	Key string `json:"key"`
	// Flags data
	Flags uint16 `json:"flag,omitempty"`
	// Exptime expire time(second)
	Exptime uint64 `json:"exp,omitempty"`
	// Cas data
	Cas uint64 `json:"cas,omitempty"`
	// Value data
	Value []byte `json:"val"`
}

type keyInfo struct {
	name    string
	nbytes  uint64
	exptime uint64
}

// NewMemcache connect memcached server
func NewMemcache(address string, dialTimeout time.Duration) (*Memcache, error) {
	var err error
	mc := Memcache{}
	mc.conn, err = youtubeMemcache.Connect(address, dialTimeout)
	return &mc, err
}

// Set data
func (mc *Memcache) Set(kv Kv) error {
	var ok bool
	var err error
	if kv.Cas != 0 {
		ok, err = mc.conn.Cas(kv.Key, kv.Flags, kv.Exptime, kv.Value, kv.Cas)
	} else {
		ok, err = mc.conn.Set(kv.Key, kv.Flags, kv.Exptime, kv.Value)
	}
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("not stored :Set key:%s", kv.Key)
	}
	return nil
}

// Close connection
func (mc *Memcache) Close() {
	mc.conn.Close()
}

// Get value
func (mc *Memcache) Get(key keyInfo) (Kv, error) {
	exptime := uint64(0)
	if t := int64(key.exptime) - time.Now().Unix(); t > 0 {
		exptime = uint64(t)
	}
	//log.Printf("now:%d key.exptime:%d exptime:%d", time.Now().Unix(), key.exptime, exptime)
	results, err := mc.conn.Get(key.name)
	if err != nil {
		log.Fatal(err)
	}
	for _, ret := range results {
		return Kv{ret.Key, ret.Flags, exptime, ret.Cas, ret.Value}, nil
	}
	return Kv{}, errors.New("Not found")
}

// Stats get status
func (mc *Memcache) Stats(params string) (reuslt []byte, err error) {
	return mc.conn.Stats(params)
}
