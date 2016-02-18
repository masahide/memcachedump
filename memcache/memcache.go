package memcache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	youtubeMemcache "github.com/youtube/vitess/go/memcache"
)

// Restore : to restore the memcached dump data
func Restore(address string, dialTimeout time.Duration) {
	dec := json.NewDecoder(os.Stdin)
	conn, err := youtubeMemcache.Connect(address, dialTimeout)
	if err != nil {
		log.Fatalf("%#v", err)
	}
	defer conn.Close()
	for {
		var kv Kv
		if err := dec.Decode(&kv); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		var ok bool
		var err error
		if kv.Cas != 0 {
			ok, err = conn.Cas(kv.Key, kv.Flags, kv.Exptime, kv.Value, kv.Cas)
		} else {
			ok, err = conn.Set(kv.Key, kv.Flags, kv.Exptime, kv.Value)
		}
		switch {
		case err != nil:
			log.Fatalf("Set key:%s, err:%s", kv.Key, err)
		case !ok:
			log.Fatalf("not stored :Set key:%s", kv.Key)
		}
		fmt.Printf("store: %#v\n", kv)
	}
}

// Stats : get stats
func Stats(address string, dialTimeout time.Duration, params string) {
	conn, err := youtubeMemcache.Connect(address, dialTimeout)
	if err != nil {
		log.Fatalf("%#v", err)
	}
	defer conn.Close()
	result, _ := conn.Stats(params)
	fmt.Printf("%s", result)
}

// List : list all keys
func List(address string, dialTimeout time.Duration) {
	conn, err := youtubeMemcache.Connect(address, dialTimeout)
	if err != nil {
		log.Fatalf("%#v", err)
	}
	defer conn.Close()
	keyCh := getListKeysChan(conn)
	for key := range keyCh {
		fmt.Println(key)
	}
}

// Dump all data
func Dump(address string, dialTimeout time.Duration) {
	conn, err := youtubeMemcache.Connect(address, dialTimeout)
	if err != nil {
		log.Fatalf("%#v", err)
	}
	defer conn.Close()
	keyCh := getListKeysChan(conn)
	getConn, err := youtubeMemcache.Connect(address, dialTimeout)
	if err != nil {
		log.Fatalf("%#v", err)
	}
	for key := range keyCh {
		results, err := getConn.Get(key.name)
		if err != nil {
			log.Fatal(err)
		}
		for _, ret := range results {
			b, err := json.Marshal(Kv{ret.Key, ret.Flags, key.exptime, ret.Cas, ret.Value})
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s\n", b)
		}
	}
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

type item struct {
	key  int
	size int
}

type keyInfo struct {
	name    string
	nbytes  uint64
	exptime uint64
}

func getListKeysChan(conn *youtubeMemcache.Connection) chan keyInfo {
	keyCh := make(chan keyInfo)
	go getListKeys(conn, keyCh)
	return keyCh
}

func getListKeys(conn *youtubeMemcache.Connection, keyCh chan keyInfo) {
	defer close(keyCh)
	itemsResult, err := conn.Stats("items")
	if err != nil {
		log.Fatal(err)
	}
	itemLines := bytes.Split(itemsResult, []byte("\n"))
	items := make([]item, 0, len(itemLines)/10)
	for _, line := range itemLines {
		var i item
		_, err := fmt.Sscanf(string(line), "STAT items:%d:number %d", &i.key, &i.size)
		if err != nil {
			continue
		}
		items = append(items, i)
	}
	for _, bucket := range items {
		result, err := conn.Stats(fmt.Sprintf("cachedump %d %d", bucket.key, bucket.size))
		if err != nil {
			log.Fatal(err)
		}
		lines := bytes.Split(result, []byte("\n"))
		//keys := make([]string, 0, len(lines))
		for _, line := range lines {
			//log.Printf("line:%#s", line)

			// https://github.com/memcached/memcached/blob/c8e357f090ec1b037ca91225a6c5565b3b9b50ef/items.c#L463
			key := keyInfo{}
			_, err := fmt.Sscanf(string(line), "ITEM %s [%d b; %d s]", &key.name, &key.nbytes, &key.exptime)
			if err != nil {
				continue
			}
			//log.Printf("key:%#v", key)
			keyCh <- key
		}
	}
}
