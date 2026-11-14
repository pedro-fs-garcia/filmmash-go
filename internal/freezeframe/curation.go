package freezeframe

import (
	"bytes"
	"filmmash/internal/film"
	"filmmash/internal/view"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type CuratedFilm struct {
	ID     int32
	Title  string
	Year   int
	Frames int
}
type CuratedFrame struct {
	ID       int32
	Path     string
	Order    int
	Approved bool
}
type CurationView struct {
	Films         []CuratedFilm
	Frames        []CuratedFrame
	SelectedID    int32
	SelectedTitle string
	CoveredDays   int
	Message       string
	Today         string
}

var tmdbImagePath = regexp.MustCompile(`^/[A-Za-z0-9]+\.(jpg|png|webp)$`)

func (h *Handler) AdminFrames(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.FormValue("film_id"), 10, 32)
	v := CurationView{SelectedID: int32(id), Today: gameDay(time.Now()).Format("2006-01-02"), Message: r.URL.Query().Get("message")}
	rows, err := h.service.repo.pool.Query(r.Context(), `SELECT f.id,f.title,f.release_year,COUNT(fr.id) FILTER(WHERE fr.approved)::int FROM films f LEFT JOIN frames fr ON fr.film_id=f.id GROUP BY f.id ORDER BY f.title`)
	if err != nil {
		http.Error(w, "Could not load curation", 500)
		return
	}
	for rows.Next() {
		var f CuratedFilm
		if err = rows.Scan(&f.ID, &f.Title, &f.Year, &f.Frames); err != nil {
			rows.Close()
			http.Error(w, "Could not load films", 500)
			return
		}
		v.Films = append(v.Films, f)
		if f.ID == int32(id) {
			v.SelectedTitle = f.Title
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		http.Error(w, "Could not load films", 500)
		return
	}
	if id > 0 {
		rows, err = h.service.repo.pool.Query(r.Context(), `SELECT id,image_path,COALESCE(curated_order,0),approved FROM frames WHERE film_id=$1 ORDER BY approved DESC,curated_order NULLS LAST,id`, int32(id))
		if err != nil {
			http.Error(w, "Could not load frames", 500)
			return
		}
		for rows.Next() {
			var f CuratedFrame
			if err = rows.Scan(&f.ID, &f.Path, &f.Order, &f.Approved); err != nil {
				rows.Close()
				http.Error(w, "Could not load frames", 500)
				return
			}
			v.Frames = append(v.Frames, f)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			http.Error(w, "Could not load frames", 500)
			return
		}
	}
	err = h.service.repo.pool.QueryRow(r.Context(), `SELECT COUNT(DISTINCT valid_at)::int FROM games g WHERE active AND valid_at >= $1 AND (SELECT COUNT(*) FROM reels r WHERE r.game_id=g.id)=5`, gameDay(time.Now())).Scan(&v.CoveredDays)
	if err != nil {
		http.Error(w, "Could not load coverage", 500)
		return
	}
	var buf bytes.Buffer
	if err = view.FreezeFrameAdmin.ExecuteTemplate(&buf, "base", v); err != nil {
		h.logger.Error("render curation", "error", err)
		http.Error(w, "Could not display curation", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
func (h *Handler) ImportFrames(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.FormValue("film_id"), 10, 32)
	if id <= 0 || h.tmdbClient == nil {
		http.Error(w, "Choose a catalogue film", 400)
		return
	}
	var exists bool
	err := h.service.repo.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM films WHERE id=$1)`, int32(id)).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Choose a catalogue film", 400)
		return
	}
	frames, err := h.tmdbClient.GetBackdrops(int32(id))
	if err != nil {
		http.Error(w, "TMDB could not provide frames. Please try again.", 502)
		return
	}
	tx, err := h.service.repo.pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Could not import frames", 500)
		return
	}
	defer tx.Rollback(r.Context())
	for _, f := range frames {
		if tmdbImagePath.MatchString(f.FilePath) && f.Width >= 780 {
			_, err = tx.Exec(r.Context(), `INSERT INTO frames(film_id,image_path) VALUES($1,$2) ON CONFLICT(image_path) DO NOTHING`, int32(id), f.FilePath)
			if err != nil {
				http.Error(w, "Could not import frames", 500)
				return
			}
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		http.Error(w, "Could not import frames", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/freezeframe?film_id=%d&message=Frames+imported.+Review+before+approving.", id), 303)
}
func (h *Handler) SaveFrames(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid curation", 400)
		return
	}
	id, _ := strconv.ParseInt(r.FormValue("film_id"), 10, 32)
	if r.FormValue("reviewed") != "yes" {
		http.Error(w, "Confirm that frames contain no titles, credits or answer-revealing text", 400)
		return
	}
	selected := map[int]int32{}
	for key, values := range r.PostForm {
		var frameID int32
		if _, err := fmt.Sscanf(key, "order_%d", &frameID); err != nil {
			continue
		}
		order, _ := strconv.Atoi(values[0])
		if order == 0 {
			continue
		}
		if order < 1 || order > 4 || selected[order] != 0 {
			http.Error(w, "Select exactly four frames with distinct positions 1–4", 400)
			return
		}
		selected[order] = frameID
	}
	if len(selected) != 4 {
		http.Error(w, "Select exactly four frames, hardest first", 400)
		return
	}
	tx, err := h.service.repo.pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Could not save curation", 500)
		return
	}
	defer tx.Rollback(r.Context())
	// Serialize edits for this film before resetting its order.
	var filmID int32
	err = tx.QueryRow(r.Context(), `SELECT id FROM films WHERE id=$1 FOR UPDATE`, int32(id)).Scan(&filmID)
	if err != nil {
		http.Error(w, "Unknown film", 400)
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE frames SET approved=false,curated_order=NULL WHERE film_id=$1`, int32(id))
	if err != nil {
		http.Error(w, "Could not save curation", 500)
		return
	}
	for order, frameID := range selected {
		tag, e := tx.Exec(r.Context(), `UPDATE frames SET approved=true,curated_order=$3 WHERE id=$1 AND film_id=$2 AND image_path ~ '^/[A-Za-z0-9]+\.(jpg|png|webp)$'`, frameID, int32(id), order)
		if e != nil || tag.RowsAffected() != 1 {
			http.Error(w, "Each selected frame must belong to this film and come from TMDB", 400)
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		http.Error(w, "Could not save curation", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/freezeframe?film_id=%d&message=Curation+saved.", id), 303)
}
func (h *Handler) ScheduleGame(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid schedule", 400)
		return
	}
	date, err := time.Parse("2006-01-02", r.FormValue("date"))
	if err != nil || date.Before(gameDay(time.Now())) {
		http.Error(w, "Choose today or a future date", 400)
		return
	}
	// Advisory lock prevents two admins scheduling the same day concurrently.
	tx, err := h.service.repo.pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Could not schedule", 500)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock($1)`, date.Unix()/86400); err != nil {
		http.Error(w, "Could not schedule", 500)
		return
	}
	var exists bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM games WHERE valid_at=$1 AND active)`, pgtype.Date{Time: date, Valid: true}).Scan(&exists); err != nil || exists {
		http.Error(w, "That day already has a game", 409)
		return
	}
	ids := r.PostForm["film_id"]
	if len(ids) != 5 {
		http.Error(w, "Choose five different curated films", 400)
		return
	}
	g := Game{ValidAt: date}
	seen := map[int32]bool{}
	for i, value := range ids {
		parsed, _ := strconv.ParseInt(value, 10, 32)
		id := int32(parsed)
		if id <= 0 || seen[id] {
			http.Error(w, "Choose five different curated films", 400)
			return
		}
		seen[id] = true
		reel := Reel{Seq: int16(i + 1)}
		if err = tx.QueryRow(r.Context(), `SELECT id,title,release_year FROM films WHERE id=$1`, id).Scan(&reel.Film.Id, &reel.Film.Title, &reel.Film.Year); err != nil {
			http.Error(w, "Unknown film", 400)
			return
		}
		rows, e := tx.Query(r.Context(), `SELECT id,image_path,curated_order FROM frames WHERE film_id=$1 AND approved ORDER BY curated_order`, id)
		if e != nil {
			http.Error(w, "Could not load frames", 500)
			return
		}
		for rows.Next() {
			var f Frame
			var order int16
			f.FilmID = id
			if e = rows.Scan(&f.ID, &f.ImagePath, &order); e != nil {
				rows.Close()
				http.Error(w, "Could not load frames", 500)
				return
			}
			reel.ReelFrames = append(reel.ReelFrames, ReelFrame{Seq: order, Difficulty: 10 - order*2, Frame: f})
		}
		e = rows.Err()
		rows.Close()
		if e != nil || len(reel.ReelFrames) != 4 {
			http.Error(w, "Each film must have four approved frames", 400)
			return
		}
		candidates, e := tx.Query(r.Context(), `SELECT id,title,release_year FROM films WHERE id<>$1 AND lower(title)<>lower($3)
 AND (imdb_id IS NULL OR imdb_id IS DISTINCT FROM (SELECT imdb_id FROM films WHERE id=$1))
 ORDER BY ABS(release_year-$2),random() LIMIT 30`, id, reel.Film.Year, reel.Film.Title)
		if e != nil {
			http.Error(w, "Could not select options", 500)
			return
		}
		var wrong []film.Film
		for candidates.Next() {
			var f film.Film
			if e = candidates.Scan(&f.Id, &f.Title, &f.Year); e != nil {
				candidates.Close()
				http.Error(w, "Could not select options", 500)
				return
			}
			wrong = append(wrong, f)
		}
		e = candidates.Err()
		candidates.Close()
		if e != nil || len(wrong) < 4 {
			http.Error(w, "At least five catalogue films are required", 400)
			return
		}
		// Prefer the closest era; randomise choices among nearby films.
		if len(wrong) > 12 {
			wrong = wrong[:12]
		}
		rand.Shuffle(len(wrong), func(a, b int) { wrong[a], wrong[b] = wrong[b], wrong[a] })
		options := append(wrong[:4:4], reel.Film)
		rand.Shuffle(len(options), func(a, b int) { options[a], options[b] = options[b], options[a] })
		for j, f := range options {
			reel.Alternatives = append(reel.Alternatives, Alternative{Seq: int16(j + 1), Film: f})
		}
		g.Reels = append(g.Reels, reel)
	}
	if err = g.Validate(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var gameID int32
	err = tx.QueryRow(r.Context(), `INSERT INTO games(valid_at) VALUES($1) RETURNING id`, pgtype.Date{Time: date, Valid: true}).Scan(&gameID)
	if err != nil {
		http.Error(w, "Could not schedule", 500)
		return
	}
	for _, reel := range g.Reels {
		var reelID int32
		if err = tx.QueryRow(r.Context(), `INSERT INTO reels(game_id,film_id,seq) VALUES($1,$2,$3) RETURNING id`, gameID, reel.Film.Id, reel.Seq).Scan(&reelID); err != nil {
			http.Error(w, "Could not schedule", 500)
			return
		}
		for _, rf := range reel.ReelFrames {
			if _, err = tx.Exec(r.Context(), `INSERT INTO reel_frames(reel_id,frame_id,difficulty,seq) VALUES($1,$2,$3,$4)`, reelID, rf.Frame.ID, rf.Difficulty, rf.Seq); err != nil {
				http.Error(w, "Could not schedule", 500)
				return
			}
		}
		for _, a := range reel.Alternatives {
			if _, err = tx.Exec(r.Context(), `INSERT INTO reel_alternatives(reel_id,film_id,seq) VALUES($1,$2,$3)`, reelID, a.Film.Id, a.Seq); err != nil {
				http.Error(w, "Could not schedule", 500)
				return
			}
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		http.Error(w, "Could not schedule", 500)
		return
	}
	http.Redirect(w, r, "/admin/freezeframe?message=Daily+game+scheduled.", 303)
}
