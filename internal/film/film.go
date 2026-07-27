package film

import (
	"filmmash/internal/tmdb"
	"math"
	"strconv"
	"strings"
)

type Director struct {
	Id   int    `db:"id"`
	Name string `db:"name"`
}

type Film struct {
	Id          int
	Title       string
	Year        int
	Director    Director
	ImagePath   string
	Popularity  float64
	VoteAverage float64
	Rating      float64
	Duelcount   int32
}

type FilmRating struct {
	Id        int
	Rating    float64
	DuelCount int32
}

type PaginationParameters struct {
	Size           int
	LastSeenId     int
	LastSeenRating float64
}

type PaginatedResponse struct {
	Films []Film
	Next  PaginationParameters
}

func ToPaginatedResponse(size int, films []Film) PaginatedResponse {
	if len(films) == 0 {
		return PaginatedResponse{}
	}
	last := films[len(films)-1]
	return PaginatedResponse{
		Films: films,
		Next: PaginationParameters{
			Size:           size,
			LastSeenId:     last.Id,
			LastSeenRating: last.Rating,
		},
	}
}

const StdRating = 1400

func FilmKFromDuelCount(duelcount int32) int16 {
	if duelcount < 20 {
		return 40 - int16(duelcount)
	}
	return 20
}

func CalculateRatings(
	winnerRating, loserRating float64,
	winnerK, loserK int16,
) (newWinnerRating, newLoserRating float64) {
	Ea := 1 / (1 + math.Pow(10, (loserRating-winnerRating)/400))
	Eb := 1 / (1 + math.Pow(10, (winnerRating-loserRating)/400))
	newWinnerRating = winnerRating + float64(winnerK)*(1-Ea)
	newLoserRating = loserRating + float64(loserK)*(0-Eb)
	return newWinnerRating, newLoserRating
}

func TMDBMovieToFilm(m tmdb.Movie) (Film, error) {
	year, _, _ := strings.Cut(m.ReleaseDate, "-")
	release_year, err := strconv.Atoi(year)
	if err != nil {
		return Film{}, err
	}
	return Film{
		Id:    m.Id,
		Title: m.Title,
		Year:  release_year,
		Director: Director{
			Id:   m.Director.Id,
			Name: m.Director.Name,
		},
		ImagePath:   m.PosterPath,
		Popularity:  float64(m.Popularity),
		VoteAverage: float64(m.VoteAverage),
	}, nil
}
