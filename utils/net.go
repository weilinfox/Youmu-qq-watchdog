package utils

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var (
	client = &http.Client{
		Transport: &http.Transport{
			Proxy: func(request *http.Request) (u *url.URL, e error) {
				return http.ProxyFromEnvironment(request)
			},
			ForceAttemptHTTP2:   true,
			MaxConnsPerHost:     0,
			MaxIdleConns:        0,
			MaxIdleConnsPerHost: 999,
		},
	}

	// UserAgent HTTP请求时使用的UA
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36 Edg/87.0.664.66"
)

// GetBytes 对给定URL发送Get请求，返回响应主体
func GetBytes(url string) ([]byte, error) {
	reader, err := HTTPGetReadCloser(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()
	return io.ReadAll(reader)
}

type gzipCloser struct {
	f io.Closer
	r *gzip.Reader
}

// NewGzipReadCloser 从 io.ReadCloser 创建 gunzip io.ReadCloser
func NewGzipReadCloser(reader io.ReadCloser) (io.ReadCloser, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	return &gzipCloser{
		f: reader,
		r: gzipReader,
	}, nil
}

// Read impls io.Reader
func (g *gzipCloser) Read(p []byte) (n int, err error) {
	return g.r.Read(p)
}

// Close impls io.Closer
func (g *gzipCloser) Close() error {
	_ = g.f.Close()
	return g.r.Close()
}

// HTTPGetReadCloser 从 Http url 获取 io.ReadCloser
func HTTPGetReadCloser(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header["User-Agent"] = []string{UserAgent}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		return NewGzipReadCloser(resp.Body)
	}
	return resp.Body, err
}
