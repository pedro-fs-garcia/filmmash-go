package freezeframe

import (
	"context"
	"errors"
	"filmmash/internal/auth"
	"filmmash/internal/database"
	"filmmash/internal/film"
	"filmmash/internal/testdb"
	"filmmash/internal/tmdb"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDailyFlowPersistenceAndAccountLimits(t *testing.T) {
	ctx := context.Background()
	pool, cleanup, err := testdb.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewService(logger, NewRepository(pool), database.NewTxManager(pool))
	h := NewHandler(logger, service)
	h.ConfigureDaily(nil, "https://example.com")
	g := Game{ValidAt: gameDay(time.Now())}
	for i := 0; i < 5; i++ {
		id := 990000 + i
		f := film.Film{Id: id, Title: fmt.Sprintf("Fixture film %d", i), Year: 2000 + i, Director: film.Director{Id: id, Name: "Fixture director"}}
		if err = film.NewRepository(pool).InsertFilm(ctx, &f); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		pool.Exec(ctx, `DELETE FROM freeze_frame_plays WHERE game_id IN(SELECT id FROM games WHERE id=$1 OR valid_at=$2 OR valid_at=$3)`, g.ID, g.ValidAt.AddDate(0, 0, -1), g.ValidAt.AddDate(0, 0, 1))
		pool.Exec(ctx, `DELETE FROM reel_frames WHERE reel_id IN(SELECT id FROM reels WHERE film_id BETWEEN 990000 AND 990004)`)
		pool.Exec(ctx, `DELETE FROM reel_alternatives WHERE reel_id IN(SELECT id FROM reels WHERE film_id BETWEEN 990000 AND 990004)`)
		pool.Exec(ctx, `DELETE FROM reels WHERE film_id BETWEEN 990000 AND 990004`)
		pool.Exec(ctx, `DELETE FROM games WHERE id=$1 OR valid_at=$2 OR valid_at=$3`, g.ID, g.ValidAt.AddDate(0, 0, -1), g.ValidAt.AddDate(0, 0, 1))
		pool.Exec(ctx, `DELETE FROM frames WHERE film_id BETWEEN 990000 AND 990004`)
		pool.Exec(ctx, `DELETE FROM films WHERE id BETWEEN 990000 AND 990004`)
		pool.Exec(ctx, `DELETE FROM directors WHERE id BETWEEN 990000 AND 990004`)
	}()
	for i := 0; i < 5; i++ {
		r := Reel{Seq: int16(i + 1), Film: film.Film{Id: 990000 + i, Title: fmt.Sprintf("Fixture film %d", i)}}
		for j := 0; j < 4; j++ {
			r.ReelFrames = append(r.ReelFrames, ReelFrame{Seq: int16(j + 1), Difficulty: int16(10 - j*2), Frame: Frame{FilmID: int32(r.Film.Id), ImagePath: fmt.Sprintf("/test%d%d.jpg", i, j)}})
		}
		for j := 0; j < 5; j++ {
			r.Alternatives = append(r.Alternatives, Alternative{Seq: int16(j + 1), Film: film.Film{Id: 990000 + j}})
		}
		g.Reels = append(g.Reels, r)
	}
	if err = service.SeedGame(ctx, &g); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := h.withPlay(ctx, g, "guest:concurrency-test", nil, func(p *Play) error { return applyTake(p, g, g.Reels[0].ID, g.Reels[0].Alternatives[0].ID, 1) })
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	accepted := 0
	for e := range results {
		if e == nil {
			accepted++
		} else if !errors.Is(e, errStaleTake) {
			t.Fatal(e)
		}
	}
	if accepted != 1 {
		t.Fatalf("concurrent take scored %d times", accepted)
	}
	authService := auth.NewService(logger, nil, auth.NewRepository(pool), database.NewTxManager(pool))
	mux := http.NewServeMux()
	mux.HandleFunc("/ui/freezeframe/", authService.SessionMiddleware(h.DailyGame))
	mux.HandleFunc("/ui/freezeframe/submit_answer", authService.SessionMiddleware(h.DailyAnswer))
	mux.HandleFunc("/ui/freezeframe/next", authService.SessionMiddleware(h.DailyNext))
	mux.HandleFunc("/ui/freezeframe/history", authService.SessionMiddleware(h.History))
	mux.HandleFunc("/ui/freezeframe/leaderboard", authService.SessionMiddleware(h.Leaderboard))
	for path, handler := range map[string]http.HandlerFunc{"/admin/freezeframe": h.AdminFrames, "/admin/freezeframe/import": h.ImportFrames, "/admin/freezeframe/curate": h.SaveFrames, "/admin/freezeframe/schedule": h.ScheduleGame} {
		mux.Handle(path, authService.SessionCtx(auth.RequiresRole("admin")(handler)))
	}
	server := httptest.NewServer(mux)
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	get := func(path string) string {
		t.Helper()
		res, e := client.Get(server.URL + path)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 {
			t.Fatalf("GET %s: %d %s", path, res.StatusCode, data)
		}
		return string(data)
	}
	post := func(path string, values url.Values) string {
		t.Helper()
		res, e := client.PostForm(server.URL+path, values)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 {
			t.Fatalf("POST %s: %d %s", path, res.StatusCode, data)
		}
		return string(data)
	}
	html := get("/ui/freezeframe/")
	if !strings.Contains(html, "Take 1 of 4") {
		t.Fatal("new player did not start")
	}
	for i, r := range g.Reels {
		// Every reel uses a different scoring path, including four failures.
		failures := i
		take := 1
		var last url.Values
		for j, a := range r.Alternatives {
			if failures == 0 {
				break
			}
			if j == i {
				continue
			}
			last = url.Values{"reel_id": {strconv.Itoa(int(r.ID))}, "alternative_id": {strconv.Itoa(int(a.ID))}, "take": {strconv.Itoa(take)}}
			html = post("/ui/freezeframe/submit_answer", last)
			failures--
			take++
			if take <= 4 {
				html = get("/ui/freezeframe/")
				if !strings.Contains(html, fmt.Sprintf("Take %d of 4", take)) {
					t.Fatal("reload lost take")
				}
				skip := post("/ui/freezeframe/next", nil)
				if !strings.Contains(skip, fmt.Sprintf("Take %d of 4", take)) {
					t.Fatal("reel skipped")
				}
			}
		}
		if i < 4 {
			last = url.Values{"reel_id": {strconv.Itoa(int(r.ID))}, "alternative_id": {strconv.Itoa(int(r.Alternatives[i].ID))}, "take": {strconv.Itoa(take)}}
			html = post("/ui/freezeframe/submit_answer", last)
			if !strings.Contains(html, "You got it.") {
				t.Fatal("correct outcome absent")
			}
			html = post("/ui/freezeframe/next", nil)
		} else {
			html = post("/ui/freezeframe/submit_answer", last)
		}
	}
	if !strings.Contains(html, "That's a wrap.") || !strings.Contains(html, ">500<span>") {
		t.Fatal("final score/results incorrect")
	}
	if !strings.Contains(get("/ui/freezeframe/"), ">500<span>") {
		t.Fatal("completed game restarted")
	}
	var userID uuid.UUID
	token := uuid.NewString()
	err = pool.QueryRow(ctx, `INSERT INTO users(idp_sub) VALUES($1) RETURNING id`, token).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, userID)
		pool.Exec(ctx, `DELETE FROM freeze_frame_plays WHERE user_id=$1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	}()
	_, err = pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,NOW()+INTERVAL '1 day')`, auth.Sha256Hash(token), userID)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := url.Parse(server.URL)
	jar.SetCookies(base, []*http.Cookie{{Name: "session", Value: token, Path: "/"}})
	if !strings.Contains(get("/ui/freezeframe/"), ">500<span>") {
		t.Fatal("login lost guest result")
	}
	secondJar, _ := cookiejar.New(nil)
	secondJar.SetCookies(base, []*http.Cookie{{Name: "session", Value: token, Path: "/"}})
	client.Jar = secondJar
	if !strings.Contains(get("/ui/freezeframe/"), ">500<span>") {
		t.Fatal("other device replayed day")
	}
	history := get("/ui/freezeframe/history")
	if !strings.Contains(history, "500 / 1000 points") || !strings.Contains(history, "Current streak: 1 days") {
		t.Fatal("account history/streak missing")
	}
	if !strings.Contains(get("/ui/freezeframe/leaderboard"), ">500</td>") {
		t.Fatal("leaderboard missing account result")
	}
	// Archive play is account-only; finishing a missed date must not repair the streak.
	past := g
	past.ID = 0
	past.ValidAt = g.ValidAt.AddDate(0, 0, -1)
	if err = service.SeedGame(ctx, &past); err != nil {
		t.Fatal(err)
	}
	day := past.ValidAt.Format("2006-01-02")
	if !strings.Contains(get("/ui/freezeframe/?date="+day), "Take 1 of 4") {
		t.Fatal("missed day not playable")
	}
	for i, r := range past.Reels {
		values := url.Values{"date": {day}, "reel_id": {strconv.Itoa(int(r.ID))}, "alternative_id": {strconv.Itoa(int(r.Alternatives[i].ID))}, "take": {"1"}}
		html = post("/ui/freezeframe/submit_answer", values)
		if i < 4 {
			html = post("/ui/freezeframe/next", url.Values{"date": {day}})
		}
	}
	if !strings.Contains(html, ">1000<span>") {
		t.Fatal("archive score wrong")
	}
	if !strings.Contains(get("/ui/freezeframe/history"), "Current streak: 1 days") {
		t.Fatal("late archive play repaired streak")
	}
	if !strings.Contains(get("/ui/freezeframe/?date="+day), "That's a wrap.") {
		t.Fatal("archive game replayable")
	}
	client.Jar, _ = cookiejar.New(nil)
	if !strings.Contains(get("/ui/freezeframe/?date="+day), "Sign in to access past games") {
		t.Fatal("guest accessed archive")
	}
	if !strings.Contains(get("/ui/freezeframe/"), "Take 1 of 4") {
		t.Fatal("second visitor did not get daily game")
	}
	res, err := client.Get(server.URL + "/admin/freezeframe")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatal("anonymous curation allowed")
	}
	client.Jar = secondJar
	res, err = client.Get(server.URL + "/admin/freezeframe")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatal("non-admin curation allowed")
	}
	if _, err = pool.Exec(ctx, `UPDATE sessions SET roles=ARRAY['admin'] WHERE user_id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	tmdbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"backdrops":[{"file_path":"/newfixture.jpg","width":1280,"height":720}]}`))
	}))
	defer tmdbServer.Close()
	h.tmdbClient = tmdb.NewClient(tmdbServer.URL, "test")
	post("/admin/freezeframe/import", url.Values{"film_id": {"990000"}})
	for _, reel := range g.Reels {
		values := url.Values{"film_id": {strconv.Itoa(reel.Film.Id)}, "reviewed": {"yes"}}
		for i, rf := range reel.ReelFrames {
			values.Set(fmt.Sprintf("order_%d", rf.Frame.ID), strconv.Itoa(4-i))
		}
		post("/admin/freezeframe/curate", values)
	}
	var first string
	if err = pool.QueryRow(ctx, `SELECT image_path FROM frames WHERE film_id=990000 AND approved AND curated_order=1`).Scan(&first); err != nil || first != g.Reels[0].ReelFrames[3].Frame.ImagePath {
		t.Fatal("curation order not saved")
	}
	future := g.ValidAt.AddDate(0, 0, 1).Format("2006-01-02")
	values := url.Values{"date": {future}}
	for _, r := range g.Reels {
		values.Add("film_id", strconv.Itoa(r.Film.Id))
	}
	if !strings.Contains(post("/admin/freezeframe/schedule", values), "Daily game scheduled.") {
		t.Fatal("curated game not scheduled")
	}
	res, err = client.PostForm(server.URL+"/admin/freezeframe/schedule", values)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatal("duplicate day scheduled")
	}
	if !strings.Contains(get("/ui/freezeframe/?date="+future), "This day is unavailable") {
		t.Fatal("future day playable")
	}
}
