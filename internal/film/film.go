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
