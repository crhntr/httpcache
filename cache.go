package httpcache

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"sync"
	"time"
)

type HTTPCache struct {
	Transport http.RoundTripper
	TTL time.Duration
	Cache sync.Map
}

type cacheOnDisk struct {
	TTL time.Duration
	Cache []struct{
		Request Request
		Record Record
	}
}

func (cache *HTTPCache) GobDecode(in []byte) error {
	var dr cacheOnDisk
	err := gob.NewDecoder(bytes.NewReader(in)).Decode(&dr)
	if err != nil {
		return err
	}
	cache.TTL = dr.TTL

	for _, value := range dr.Cache {
		cache.Cache.Store(value.Request, value.Record)
	}

	return nil
}

func (cache *HTTPCache) GobEncode() ([]byte, error) {
	var buf bytes.Buffer

	var records []struct{
		Request Request
		Record Record
	}

	cache.Cache.Range(func(key, value interface{}) bool {
		records = append(records, struct{
			Request Request
			Record Record
		}{
			Request: key.(Request),
			Record: value.(Record),
		})
		return true
	})

	err := gob.NewEncoder(&buf).Encode(cacheOnDisk{
		TTL: cache.TTL,
		Cache: records,
	})
	return buf.Bytes(), err
}

func (cache *HTTPCache) Save(writer io.Writer) error {
	return gob.NewEncoder(writer).Encode(cache)
}

func (cache *HTTPCache) Load(reader io.Reader) error {
	return gob.NewDecoder(reader).Decode(cache)
}

func (cache *HTTPCache) SaveToFile(fp string) (retErr error) {
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			_ = f.Close()
			return
		}
		retErr = f.Close()
	}()
	return cache.Save(f)
}

func (cache *HTTPCache) LoadFromFile(fp string) (retErr error) {
	f, err := os.Open(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() {
		if retErr != nil {
			_ = f.Close()
			return
		}
		retErr = f.Close()
	}()
	return cache.Load(f)
}

type Request struct {
	Method 		string
	URL    		string
	HeadersHash string
}

type Record struct{
	Timestamp time.Time
	Request []byte
	Response []byte
}

func (record Record) GetResponse() (*http.Response, error) {
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(record.Request)))
	if err != nil {
		return nil, err
	}
	res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(record.Response)), req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

var _ http.RoundTripper = (*HTTPCache)(nil)

func (cache *HTTPCache) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	if cache.Transport == nil {
		cache.Transport = http.DefaultTransport
	}

	sum := sha256.New()
	_ = httpRequest.Header.Write(sum)

	key := Request{URL: httpRequest.URL.String(), Method: httpRequest.Method, HeadersHash: fmt.Sprintf("%x", sum.Sum(nil))}
	if value, found := cache.Cache.Load(key); found && (cache.TTL == 0 || time.Since(value.(Record).Timestamp) < cache.TTL) {
		return value.(Record).GetResponse()
	}

	httpResponse, err := cache.Transport.RoundTrip(httpRequest)
	if err != nil {
		return httpResponse, err
	}

	var record Record

	record.Timestamp = time.Now()

	record.Request, err = httputil.DumpRequest(httpRequest, true)
	if err != nil {
		return httpResponse, err
	}

	record.Response, err = httputil.DumpResponse(httpResponse, true)
	if err != nil {
		return httpResponse, err
	}

	if httpResponse.StatusCode >= 200 && httpResponse.StatusCode < 300 {
		cache.Cache.Store(key, record)
	}

	return record.GetResponse()
}
