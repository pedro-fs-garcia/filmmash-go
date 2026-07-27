package tmdb

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) GetPopulars(page int16) ([]Movie, error) {
	path := "/movie/popular?language=en-US&page=" + strconv.Itoa(int(page))
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

func (c *Client) GetDirector(movieId int) (*Director, error) {
	str := fmt.Sprintf("/movie/%d/credits", movieId)
	req, err := c.newRequest("GET", str, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var result CreditsResponse
	err = c.Do(req, &result)
	if err != nil {
		log.Println(err)
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

func (c *Client) GetTopRated(page int16) ([]Movie, error) {
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

	movies := result.Results

	for i := range movies {
		dir, _ := c.GetDirector(movies[i].Id)
		if dir != nil {
			movies[i].Director = *dir
		}
	}
	return movies, nil
}

func (c *Client) SearchFilm(params *SearchFilmParams) (*TmdbStandardResponse, error) {
	u, err := url.Parse("/search/movie")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("query", params.Query)
	if params.Page > 0 {
		q.Set("page", strconv.Itoa(int(params.Page)))
	}
	u.RawQuery = q.Encode()

	req, err := c.newRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	var result TmdbStandardResponse
	err = c.Do(req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

type FilmDetailResponse struct {
	Id          int             `json:"id"`
	Title       string          `json:"title"`
	ReleaseDate string          `json:"release_date"`
	PosterPath  string          `json:"poster_path"`
	VoteAverage float32         `json:"vote_average"`
	Popularity  float32         `json:"popularity"`
	Credits     CreditsResponse `json:"credits"`
}

func (c *Client) GetFilm(filmId int32) (Movie, error) {
	path := fmt.Sprintf("/movie/%d?append_to_response=credits", filmId)
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return Movie{}, err
	}

	var result FilmDetailResponse
	err = c.Do(req, &result)
	if err != nil {
		return Movie{}, err
	}

	var dir Director
	for _, c := range result.Credits.Crew {
		if c.Job == "Director" {
			dir.Id = c.Id
			dir.Name = c.Name
			break
		}
	}

	return Movie{
		Id:          result.Id,
		Title:       result.Title,
		ReleaseDate: result.ReleaseDate,
		PosterPath:  result.PosterPath,
		VoteAverage: result.VoteAverage,
		Popularity:  result.Popularity,
		Director:    dir,
	}, nil
}
