package main

import (
	"fmt"
	"log"
	"mycache/cache"
	myhttp "mycache/http"
	"net/http"
)

// 模拟 db 数据
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	cache.NewGroup("scores", 2<<10, cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[MyDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	addr := "localhost:9999"
	peers := myhttp.NewPool(addr)
	log.Println("mycache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
