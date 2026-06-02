package tmdb

import (
	"fmt"
	"log"
	"strconv"
)

type Movie struct {
	Id          int     `json:"id"`
	Title       string  `json:"title"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float32 `json:"vote_average"`
	Director    Director
}

type Director struct {
	Id   int
	Name string
}

type MovieCredits struct {
	Job  string `json:"job"`
	Name string `json:"name"`
	Id   int    `json:"id"`
}

type CreditsResponse struct {
	Crew []MovieCredits `json:"crew"`
}

type TmdbStandardResponse struct {
	Page    int     `json:"page"`
	Results []Movie `json:"results"`
}

func (c *TmdbClient) GetPopulars(page int) ([]Movie, error) {
	path := "/movie/popular?language=en-US&page=" + strconv.Itoa(page)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var result TmdbStandardResponse
	err = c.Do(req, &result)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	movies := result.Results

	for i := range movies {
		dir, _ := c.GetDirector(movies[i].Id)
		if dir != nil {
			movies[i].Director = *dir
		}
	}

	return movies, nil
}

func (c *TmdbClient) GetDirector(movieId int) (*Director, error) {
	str := fmt.Sprintf("/movie/%d/credits", movieId)
	req, err := c.newRequest("GET", str, nil)
	if err != nil {
		return nil, err
	}

	var result CreditsResponse
	err = c.Do(req, &result)
	if err != nil {
		return nil, err
	}

	for i := range result.Crew {
		if result.Crew[i].Job == "Director" {
			d := result.Crew[i]
			return &Director{
				Id:   d.Id,
				Name: d.Name,
			}, nil
		}
	}

	return nil, nil
}

func (c *TmdbClient) GetTopRated(page int) ([]Movie, error) {
	var total int
	movies := make([]Movie, 0)

	path := fmt.Sprintf("/movie/top_rated?language=en-US&page=%d", page)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result TmdbStandardResponse
	err = c.Do(req, &result)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	movies = append(movies, result.Results...)

	total += len(result.Results)
	page++

	for i := range movies {
		dir, _ := c.GetDirector(movies[i].Id)
		if dir != nil {
			movies[i].Director = *dir
		}
	}
	return movies, nil
}

// func (c *TmdbClient) GetPosterImage(imgRef string) {
// 	url := fmt.Sprintf("%s/t/p/w500/%s", c.baseURL, imgRef)

// }
