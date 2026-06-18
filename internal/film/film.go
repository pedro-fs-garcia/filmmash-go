package film

type Director struct {
	Id   int    `db:"id"`
	Name string `db:"name"`
}

type Film struct {
	Id        int
	Title     string
	Year      int
	Director  Director
	ImagePath string
	Rating    float64
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
