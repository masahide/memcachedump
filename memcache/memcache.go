package memcache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// Restore : to restore the memcached dump data
func Restore(address string, dialTimeout time.Duration) error {
	dec := json.NewDecoder(os.Stdin)
	conn, err := NewMemcache(address, dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		var kv Kv
		if err = dec.Decode(&kv); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		if err = conn.Set(kv); err != nil {
			return err
		}
	}
}

// Stats : get stats
func Stats(address string, dialTimeout time.Duration, params string) ([]byte, error) {
	conn, err := NewMemcache(address, dialTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.Stats(params)
}

// PrintList : list all keys
func PrintList(address string, dialTimeout time.Duration) error {
	conn, err := NewMemcache(address, dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	keyCh := getListKeysChan(conn)
	for key := range keyCh {
		fmt.Println(key)
	}
	return nil
}

// PrintDump all data
func PrintDump(address string, dialTimeout time.Duration) error {
	conn, err := NewMemcache(address, dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	keyCh := getListKeysChan(conn)
	getConn, err := NewMemcache(address, dialTimeout)
	if err != nil {
		log.Fatalf("%#v", err)
	}
	for key := range keyCh {
		result, err := getConn.Get(key)
		if err != nil {
			return err
		}
		b, err := json.Marshal(result)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", b)
	}
	return nil
}

type getStats interface {
	Stats(argument string) (result []byte, err error)
}

func getListKeysChan(conn getStats) chan keyInfo {
	keyCh := make(chan keyInfo)
	go getListKeys(conn, keyCh)
	return keyCh
}

type item struct {
	key  int
	size int
}

func getListKeys(conn getStats, keyCh chan keyInfo) {
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
