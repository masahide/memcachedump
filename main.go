package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/youtube/vitess/go/cacheservice"
	"github.com/youtube/vitess/go/memcache"
)

var (
	address     = "localhost:11211"
	dialTimeout = 10 * time.Second
)

func main() {
	flag.StringVar(&address, "address", address, "memcached address")
	flag.DurationVar(&dialTimeout, "timeout", dialTimeout, "dial timeout")
	flag.Parse()
	args := flag.Args()
	action := ""

	if len(args) > 0 {
		action = args[0]
	}
	switch action {
	case "stats":
		params := ""
		if len(args) > 1 {
			params = strings.Join(args[1:], " ")
		}
		stats(address, dialTimeout, params)
	case "list":
		list(address, dialTimeout)
	case "dump":
		dump(address, dialTimeout)
	case "restore":
		restore(address, dialTimeout)
	default:
		fmt.Println(`usage:

 $ memcached -address 10.0.0.1:11211 dump > dump.json
 $ memcached -address 10.0.0.2:11211 restore < dump.json
 $ memcached -address 10.0.0.1:11211 dump | memcached -address 10.0.0.2:11211 restore
`)
	}
}

func restore(address string, dialTimeout time.Duration) {
	dec := json.NewDecoder(os.Stdin)
	conn, err := memcache.Connect(address, dialTimeout)
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
		if m.Cas != 0 {
			ok, err := conn.Cas(m.Key, m.Flags, 0, m.Value, m.Cas)
			if err != nil {
				log.Fatalf("Cas key:%s, err:%s", m.Key, err)
			}
			if !ok {
				log.Fatalf("not stored :Cas key:%s", m.Key)
			}
		} else {
			ok, err := conn.Set(m.Key, m.Flags, 0, m.Value)
			if err != nil {
				log.Fatalf("Set key:%s, err:%s", m.Key, err)
			}
			if !ok {
				log.Fatalf("not stored :Set key:%s", m.Key)
			}
		}
		log.Printf("store: %#v", m)
	}
}

func stats(address string, dialTimeout time.Duration, params string) {
	conn, err := memcache.Connect(address, dialTimeout)
	if err != nil {
		log.Fatalf("%#v", err)
	}
	defer conn.Close()
	result, _ := conn.Stats(params)
	fmt.Printf("%s", result)
}

func list(address string, dialTimeout time.Duration) {
	conn, err := memcache.Connect(address, dialTimeout)
	if err != nil {
		log.Fatalf("%#v", err)
	}
	defer conn.Close()
	keyCh := getListKeys(conn)
	for key := range keyCh {
		fmt.Println(key)
	}
}

func dump(address string, dialTimeout time.Duration) {
	conn, err := memcache.Connect(address, dialTimeout)
	if err != nil {
		log.Fatalf("%#v", err)
	}
	defer conn.Close()
	keyCh := getListKeys(conn)
	getConn, err := memcache.Connect(address, dialTimeout)
	if err != nil {
		log.Printf("%#v", err)
		os.Exit(1)
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
			os.Stdout.Write(b)
			os.Stdout.Write([]byte("\n"))
		}

	}
}

type Item struct {
	Key  int
	Size int
}

func getListKeys(conn *memcache.Connection) chan string {
	keyCh := make(chan string)
	go func() {
		defer close(keyCh)
		itemsResult, err := conn.Stats("items")
		if err != nil {
			log.Fatal(err)
		}
		itemLines := bytes.Split(itemsResult, []byte("\n"))
		items := make([]Item, 0, len(itemLines)/10)
		for _, line := range itemLines {
			var item Item
			_, err := fmt.Sscanf(string(line), "STAT items:%d:number %d", &item.Key, &item.Size)
			if err != nil {
				continue
			}
			items = append(items, item)
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
	}()
	return keyCh
}
