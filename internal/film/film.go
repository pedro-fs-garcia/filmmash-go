package film

import "math"

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
	Id     int
	Rating float64
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

func CalculateRatings(winnerRating, loserRating float64) (newWinnerRating, newLoserRating float64) {
	const K = 20
	Ea := 1 / (1 + math.Pow(10, (loserRating-winnerRating)/400))
	Eb := 1 / (1 + math.Pow(10, (winnerRating-loserRating)/400))
	newWinnerRating = winnerRating + K*(1-Ea)
	newLoserRating = loserRating + K*(0-Eb)
	return newWinnerRating, newLoserRating
}
