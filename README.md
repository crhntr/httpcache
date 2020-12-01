# httpcache

Example

```go

func main() {
	const cacheFileName = "/tmp/http_cache"
	cache := new(HTTPCache)
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
		fmt.Println(time.Since(last), res.StatusCode)
		last = time.Now()
	}
}

```
