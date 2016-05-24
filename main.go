package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/masahide/memcachedump/memcache"
)

var (
	address     = "localhost:11211"
	dialTimeout = 10 * time.Second
	usage       = `usage:

 $ memcachedump -address 10.0.0.1:11211 list 
 $ memcachedump -address 10.0.0.1:11211 dump > dump.json
 $ memcachedump -address 10.0.0.2:11211 restore < dump.json
 $ memcachedump -address 10.0.0.1:11211 dump | memcachedump -address 10.0.0.2:11211 restore
`
)

func main() {
	flag.StringVar(&address, "address", address, "memcached address")
	flag.DurationVar(&dialTimeout, "timeout", dialTimeout, "dial timeout")
	flag.Parse()
	args := flag.Args()
	action := ""

	if len(args) >= 1 {
		action = args[0]
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	var err error
	switch action {
	case "stats":
		params := ""
		if len(args) > 1 {
			params = strings.Join(args[1:], " ")
		}
		result, serr := memcache.Stats(address, dialTimeout, params)
		if serr != nil {
			log.Fatal(serr)
		}
		fmt.Printf("%s\n", result)
	case "list":
		err = memcache.PrintList(address, dialTimeout)
	case "dump":
		err = memcache.PrintDump(address, dialTimeout)
	case "restore":
		err = memcache.Restore(address, dialTimeout)
	default:
		flag.PrintDefaults()
		fmt.Println(usage)
	}
	if err != nil {
		log.Fatal(err)
	}
}
