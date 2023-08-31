package feed

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Downloader struct {
}

func NewDownloader() *Downloader {
	return &Downloader{}
}

func (*Downloader) Download(url string) (io.ReadCloser, error) {
	client := http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "rsslay")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("http error %d", resp.StatusCode)
	}

	return resp.Body, nil
}
