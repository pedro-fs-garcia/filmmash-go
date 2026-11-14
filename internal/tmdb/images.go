package tmdb

import "fmt"

type Backdrop struct {
	FilePath string `json:"file_path"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

func (c *Client) GetBackdrops(filmID int32) ([]Backdrop, error) {
	req, err := c.newRequest("GET", fmt.Sprintf("/movie/%d/images?include_image_language=null", filmID), nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Backdrops []Backdrop `json:"backdrops"`
	}
	if err = c.Do(req, &response); err != nil {
		return nil, err
	}
	return response.Backdrops, nil
}
