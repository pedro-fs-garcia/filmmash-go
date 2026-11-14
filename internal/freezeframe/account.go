package freezeframe

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"filmmash/internal/view"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"net/http"
	"time"
)

type ArchiveDay struct {
	Date     string
	Score    int
	Finished bool
	Started  bool
	Outcomes []Outcome
}
type Leader struct {
	Rank  int
	Name  string
	Score int
	Games int
}
type AccountView struct {
	Title       string
	Days        []ArchiveDay
	Leaders     []Leader
	Streak      int
	Date        string
	Leaderboard bool
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireAccount(w, r)
	if !ok {
		return
	}
	if _, _, err := h.player(w, r); err != nil {
		http.Error(w, "Could not load history", 500)
		return
	}
	rows, err := h.service.repo.pool.Query(r.Context(), `SELECT g.valid_at,COALESCE(p.score,0),COALESCE(p.finished,false),p.id IS NOT NULL,COALESCE(p.progress,'{}'::jsonb),p.finished_at
 FROM games g LEFT JOIN freeze_frame_plays p ON p.game_id=g.id AND p.user_id=$1
 WHERE g.active AND g.valid_at <= $2
 ORDER BY g.valid_at DESC`, session.UserID, gameDay(time.Now()))
	if err != nil {
		http.Error(w, "Could not load history", 500)
		return
	}
	defer rows.Close()
	v := AccountView{Title: "Your screenings"}
	eligible := map[string]bool{}
	for rows.Next() {
		var d pgtype.Date
		var day ArchiveDay
		var data []byte
		var finishedAt *time.Time
		if err = rows.Scan(&d, &day.Score, &day.Finished, &day.Started, &data, &finishedAt); err != nil {
			http.Error(w, "Could not load history", 500)
			return
		}
		day.Date = d.Time.Format("2006-01-02")
		var progress PlayProgress
		if err = json.Unmarshal(data, &progress); err != nil {
			http.Error(w, "Could not load history", 500)
			return
		}
		day.Outcomes = progress.Outcomes
		v.Days = append(v.Days, day)
		if day.Finished && finishedAt != nil && gameDay(*finishedAt).Equal(d.Time) {
			eligible[day.Date] = true
		}
	}
	if rows.Err() != nil {
		http.Error(w, "Could not load history", 500)
		return
	}
	day := gameDay(time.Now())
	if !eligible[day.Format("2006-01-02")] {
		day = day.AddDate(0, 0, -1)
	}
	for eligible[day.Format("2006-01-02")] {
		v.Streak++
		day = day.AddDate(0, 0, -1)
	}
	h.renderAccount(w, r, v)
}
func (h *Handler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAccount(w, r); !ok {
		return
	}
	date := gameDay(time.Now())
	overall := r.URL.Query().Get("period") == "overall"
	rows, err := h.service.repo.pool.Query(r.Context(), `SELECT p.user_id::text,SUM(p.score)::int,COUNT(*)::int FROM freeze_frame_plays p JOIN games g ON g.id=p.game_id
 WHERE g.active AND p.user_id IS NOT NULL AND p.finished AND ($2 OR g.valid_at=$1)
 GROUP BY p.user_id ORDER BY SUM(p.score) DESC,MIN(p.finished_at),p.user_id LIMIT 100`, date, overall)
	if err != nil {
		http.Error(w, "Could not load leaderboard", 500)
		return
	}
	defer rows.Close()
	v := AccountView{Title: "Today's leaderboard", Date: date.Format("2006-01-02"), Leaderboard: true}
	if overall {
		v.Title = "Overall leaderboard"
	}
	for rows.Next() {
		var id string
		var entry Leader
		if err = rows.Scan(&id, &entry.Score, &entry.Games); err != nil {
			http.Error(w, "Could not load leaderboard", 500)
			return
		}
		entry.Rank = len(v.Leaders) + 1
		// A stable alias avoids exposing internal account ids or private profile data.
		digest := sha256.Sum256([]byte(id))
		entry.Name = fmt.Sprintf("Film fan %x", digest[:4])
		v.Leaders = append(v.Leaders, entry)
	}
	if rows.Err() != nil {
		http.Error(w, "Could not load leaderboard", 500)
		return
	}
	h.renderAccount(w, r, v)
}
func (h *Handler) renderAccount(w http.ResponseWriter, r *http.Request, v AccountView) {
	var buf bytes.Buffer
	if err := view.FreezeFrameAccount.ExecuteTemplate(&buf, "base", v); err != nil {
		h.logger.Error("render game account", "error", err)
		http.Error(w, "Could not display this page", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
