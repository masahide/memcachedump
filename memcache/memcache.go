package memcache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/youtube/vitess/go/cacheservice"
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
		var m cacheservice.Result
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		var ok bool
		var err error
		if m.Cas != 0 {
			ok, err = conn.Cas(m.Key, m.Flags, 0, m.Value, m.Cas)
		} else {
			ok, err = conn.Set(m.Key, m.Flags, 0, m.Value)
		}
		switch {
		case err != nil:
			log.Fatalf("Set key:%s, err:%s", m.Key, err)
		case !ok:
			log.Fatalf("not stored :Set key:%s", m.Key)
		}
		log.Printf("store: %#v", m)
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
		results, err := getConn.Get(key)
		if err != nil {
			log.Fatal(err)
		}
		for _, ret := range results {
			b, err := json.Marshal(ret)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s\n", b)
		}

	}
}

type item struct {
	Key  int
	Size int
}

func getListKeysChan(conn *youtubeMemcache.Connection) chan string {
	keyCh := make(chan string)
	go getListKeys(conn, keyCh)
	return keyCh
}

func getListKeys(conn *youtubeMemcache.Connection, keyCh chan string) {
	defer close(keyCh)
	itemsResult, err := conn.Stats("items")
	if err != nil {
		log.Fatal(err)
	}
	itemLines := bytes.Split(itemsResult, []byte("\n"))
	items := make([]item, 0, len(itemLines)/10)
	for _, line := range itemLines {
		var i item
		_, err := fmt.Sscanf(string(line), "STAT items:%d:number %d", &i.Key, &i.Size)
		if err != nil {
			continue
		}
		items = append(items, i)
	}
	for _, bucket := range items {
		result, err := conn.Stats(fmt.Sprintf("cachedump %d %d", bucket.Key, bucket.Size))
		if err != nil {
			log.Fatal(err)
		}
		lines := bytes.Split(result, []byte("\r\n"))
		//keys := make([]string, 0, len(lines))
		for _, line := range lines {
			words := bytes.Split(line, []byte(" "))
			if len(words) < 2 {
				continue
			}
			keyCh <- string(words[1])
		}
	}
}
