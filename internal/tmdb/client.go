package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TmdbClient struct {
	baseURL     string
	accessToken string
	http        *http.Client
}

func NewTmdbClient() *TmdbClient {
	return &TmdbClient{
		baseURL:     "https://api.themoviedb.org/3",
		accessToken: "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiI0NzU4NTFhN2I5ZGFlNjIwZTE4N2RlOGE3N2ZiZGFiYyIsIm5iZiI6MTczNzIxMjUxMS42NjUsInN1YiI6IjY3OGJjMjVmMDhkZDcwOGJiOTZkZTAwNSIsInNjb3BlcyI6WyJhcGlfcmVhZCJdLCJ2ZXJzaW9uIjoxfQ.rabzs1fy3QwieG2IGs9Yh-d1z9DmJguI8rRV2evv770",
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *TmdbClient) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.accessToken)
	return req, nil
}

func (c *TmdbClient) Do(req *http.Request, out any) error {
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
