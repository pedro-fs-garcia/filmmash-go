package tmdb

type Movie struct {
	Id          int     `json:"id"`
	Title       string  `json:"title"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float32 `json:"vote_average"`
	Popularity  float32 `json:"popularity"`
	Director    Director
	InCatalogue bool
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
	Page         int     `json:"page"`
	TotalPages   int     `json:"total_pages"`
	TotalResults int     `json:"total_results"`
	Results      []Movie `json:"results"`
}

type SearchFilmParams struct {
	Query        string `json:"query"`
	IncludeAdult bool   `json:"include_adult"`
	Language     string `json:"language"`
	ReleseYear   string `json:"primary_release_year"`
	Page         int32  `json:"Page"`
	Region       string `json:"region"`
	Year         string `json:"year"`
}
