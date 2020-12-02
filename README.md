# httpcache

Example

```go

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/crhntr/httpcache"
)

func main() {
	const cacheFileName = "/tmp/http_cache"
	cache := new(httpcache.HTTPCache)
	if err := cache.LoadFromFile(cacheFileName); err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := cache.SaveToFile(cacheFileName); err != nil {
			log.Fatal(err)
		}
	}()

	http.DefaultClient.Transport = cache

	req, _ := http.NewRequest(http.MethodGet, "https://crhntr.com/ip", nil)

	last := time.Now()
	for i := 0; i < 10; i++ {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println(err)
			continue
		}
		_, _ = io.Copy(os.Stdout, res.Body)
		_ = res.Body.Close()
		fmt.Println(time.Since(last), res.StatusCode)
		last = time.Now()
	}
}

```
