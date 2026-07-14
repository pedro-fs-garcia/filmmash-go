package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL     string
	accessToken string
	http        *http.Client
}

func NewClient(baseUrl string, accessToken string) *Client {
	return &Client{
		baseURL:     baseUrl,
		accessToken: accessToken,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.accessToken)
	return req, nil
}

func (c *Client) Do(req *http.Request, out any) error {
	res, err := c.http.Do(req)
	if err != nil {
		fmt.Println("Error accessing tmdb")
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("TMDB: %d %s", res.StatusCode, body)
	}
	return json.NewDecoder(res.Body).Decode(out)
}
