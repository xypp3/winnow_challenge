package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ContentItem struct {
	Unavailable bool   `json:"unavailable"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
	ExpiresAt   string `json:"expiresAt"`
	ETag        string `json:"ETag"`
}

type ManifestItem struct {
	Unavailable bool          `json:"unavailable"`
	Items       []ContentItem `json:"items"`
}

type Response struct {
	ETag        string
	StatusCode  int
	ContentType string
	Manifest    map[string]ManifestItem
}

type Client struct {
	baseURL     string
	deviceToken string
	httpClient  *http.Client
}

func NewClient(baseURL, deviceToken string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:     baseURL,
		deviceToken: deviceToken,
		httpClient:  httpClient,
	}
}

func (c *Client) Fetch(ctx context.Context, etag string) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return Response{}, err
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if c.deviceToken != "" {
		req.Header.Set("X-Authorization-Device", c.deviceToken)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer res.Body.Close()

	responseETag := res.Header.Get("ETag")

	switch res.StatusCode {
	case http.StatusNotModified:
		return Response{
			StatusCode: res.StatusCode,
			ETag:       responseETag,
		}, nil
	case http.StatusOK:
		manifest, err := decodeManifest(res.Body)
		if err != nil {
			return Response{}, err
		}
		return Response{
			StatusCode:  res.StatusCode,
			ETag:        responseETag,
			Manifest:    manifest,
			ContentType: res.Header.Get("Content-Type"),
		}, nil
	default:
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return Response{}, fmt.Errorf("unexpected manifest response %d: %s", res.StatusCode, string(body))
	}
}

func decodeManifest(body io.Reader) (map[string]ManifestItem, error) {
	var manifest map[string]ManifestItem
	if err := json.NewDecoder(body).Decode(&manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}
