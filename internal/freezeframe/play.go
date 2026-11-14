package freezeframe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filmmash/internal/auth"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"net/url"
	"time"
)

// Every player uses the same UTC calendar boundary, including the archive.
func gameDay(now time.Time) time.Time {
	n := now.UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

type Outcome struct {
	Reel    int    `json:"reel"`
	Film    string `json:"film"`
	Correct bool   `json:"correct"`
	Takes   int    `json:"takes"`
	Points  int    `json:"points"`
}
type PlayProgress struct {
	Current  int       `json:"current"`
	Guesses  []int32   `json:"guesses"`
	Outcomes []Outcome `json:"outcomes"`
	Pending  bool      `json:"pending"`
}
type Play struct {
	ID       uuid.UUID
	Progress PlayProgress
	Score    int
	Finished bool
}

var errStaleTake = errors.New("this take has already been submitted; refresh to resume")

// applyTake never trusts client score, correctness, take number or frame count.
func applyTake(p *Play, g Game, reelID, alternativeID int32, take int) error {
	if p.Finished || p.Progress.Pending || p.Progress.Current < 0 || p.Progress.Current >= len(g.Reels) {
		return errStaleTake
	}
	reel := g.Reels[p.Progress.Current]
	if reel.ID != reelID || take != len(p.Progress.Guesses)+1 {
		return errStaleTake
	}
	for _, id := range p.Progress.Guesses {
		if id == alternativeID {
			return errStaleTake
		}
	}
	var chosen *Alternative
	for i := range reel.Alternatives {
		if reel.Alternatives[i].ID == alternativeID {
			chosen = &reel.Alternatives[i]
			break
		}
	}
	if chosen == nil {
		return errors.New("choose an option from the current reel")
	}
	p.Progress.Guesses = append(p.Progress.Guesses, alternativeID)
	correct := chosen.Film.Id == reel.Film.Id
	if correct || len(p.Progress.Guesses) == 4 {
		points := 0
		if correct {
			points = 250 - 50*len(p.Progress.Guesses)
		}
		p.Score += points
		p.Progress.Outcomes = append(p.Progress.Outcomes, Outcome{Reel: p.Progress.Current + 1, Film: reel.Film.Title, Correct: correct, Takes: len(p.Progress.Guesses), Points: points})
		p.Progress.Pending = true
		p.Finished = p.Progress.Current == len(g.Reels)-1
	}
	return nil
}

func (h *Handler) player(w http.ResponseWriter, r *http.Request) (string, *uuid.UUID, error) {
	cookie, err := r.Cookie("freeze_frame_player")
	token := uuid.Nil
	if err == nil {
		token, _ = uuid.Parse(cookie.Value)
	}
	if token == uuid.Nil {
		token, err = uuid.NewRandom()
		if err != nil {
			return "", nil, err
		}
		http.SetCookie(w, &http.Cookie{Name: "freeze_frame_player", Value: token.String(), Path: "/", HttpOnly: true, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", SameSite: http.SameSiteLaxMode, MaxAge: 365 * 86400})
	}
	digest := sha256.Sum256([]byte(token.String()))
	guest := "guest:" + hex.EncodeToString(digest[:])
	if session, ok := auth.SessionFromContext(r.Context()); ok {
		key := "user:" + session.UserID.String()
		// Adopt this device's anonymous games without replacing an account's existing play.
		_, err = h.service.repo.pool.Exec(r.Context(), `INSERT INTO freeze_frame_plays(game_id,player_key,user_id,progress,score,finished,started_at,finished_at)
   SELECT game_id,$1,$2,progress,score,finished,started_at,finished_at FROM freeze_frame_plays WHERE player_key=$3
   ON CONFLICT(game_id,player_key) DO UPDATE SET
    progress=EXCLUDED.progress,score=EXCLUDED.score,finished=EXCLUDED.finished,
    started_at=EXCLUDED.started_at,finished_at=EXCLUDED.finished_at
   WHERE NOT freeze_frame_plays.finished AND (
    EXCLUDED.finished OR
    COALESCE(jsonb_array_length(EXCLUDED.progress->'outcomes'),0)*10 + COALESCE(jsonb_array_length(EXCLUDED.progress->'guesses'),0) >
    COALESCE(jsonb_array_length(freeze_frame_plays.progress->'outcomes'),0)*10 + COALESCE(jsonb_array_length(freeze_frame_plays.progress->'guesses'),0))`, key, session.UserID, guest)
		return key, &session.UserID, err
	}
	return guest, nil, nil
}

func (h *Handler) dailyGame(ctx context.Context, r *http.Request) (Game, error) {
	day := gameDay(time.Now())
	if value := r.FormValue("date"); value != "" {
		requested, err := time.Parse("2006-01-02", value)
		if err != nil {
			return Game{}, errors.New("invalid game date")
		}
		if requested.After(day) {
			return Game{}, errors.New("that game is not available yet")
		}
		if !requested.Equal(day) {
			if _, ok := auth.SessionFromContext(ctx); !ok {
				return Game{}, errors.New("sign in to play the archive")
			}
		}
		day = requested
	}
	ids, err := h.service.repo.GetGameIDsByDate(ctx, day)
	if err != nil {
		return Game{}, err
	}
	if len(ids) == 0 {
		return Game{}, ErrNoGameForDate
	}
	g, err := h.service.repo.GetGame(ctx, ids[0])
	if err != nil {
		return Game{}, err
	}
	if len(g.Reels) != 5 {
		return Game{}, errors.New("today's game is still being prepared")
	}
	for _, reel := range g.Reels {
		if len(reel.ReelFrames) < 4 || len(reel.Alternatives) != 5 {
			return Game{}, errors.New("today's game is still being prepared")
		}
	}
	return g, nil
}

func (h *Handler) withPlay(ctx context.Context, g Game, key string, userID *uuid.UUID, change func(*Play) error) (Play, error) {
	tx, err := h.service.repo.pool.Begin(ctx)
	if err != nil {
		return Play{}, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO freeze_frame_plays(game_id,player_key,user_id) VALUES($1,$2,$3) ON CONFLICT(game_id,player_key) DO NOTHING`, g.ID, key, userID)
	if err != nil {
		return Play{}, err
	}
	var p Play
	var data []byte
	err = tx.QueryRow(ctx, `SELECT id,progress,score,finished FROM freeze_frame_plays WHERE game_id=$1 AND player_key=$2 FOR UPDATE`, g.ID, key).Scan(&p.ID, &data, &p.Score, &p.Finished)
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal(data, &p.Progress); err != nil {
		return p, err
	}
	if change != nil {
		if err = change(&p); err != nil {
			return p, err
		}
		data, err = json.Marshal(p.Progress)
		if err != nil {
			return p, err
		}
		_, err = tx.Exec(ctx, `UPDATE freeze_frame_plays SET progress=$2,score=$3,finished=$4,finished_at=CASE WHEN $4 THEN COALESCE(finished_at,NOW()) ELSE NULL END WHERE id=$1`, p.ID, data, p.Score, p.Finished)
		if err != nil {
			return p, err
		}
	}
	return p, tx.Commit(ctx)
}

type DailyOption struct {
	ID         int32
	Key, Title string
	Year       int
	Wrong      bool
}
type DailyView struct {
	Date     string
	GameID   int32
	ReelID   int32
	Reel     int
	Take     int
	Score    int
	ImageURL string
	Options  []DailyOption
	Outcomes []Outcome
	Outcome  *Outcome
	Finished bool
	Pending  bool
	SignedIn bool
	NextAt   int64
	ShareURL string
	Error    string
}

func dailyView(g Game, p Play, signedIn bool) DailyView {
	v := DailyView{Date: g.ValidAt.Format("2006-01-02"), GameID: g.ID, Reel: p.Progress.Current + 1, Take: len(p.Progress.Guesses) + 1, Score: p.Score, Outcomes: p.Progress.Outcomes, Finished: p.Finished, Pending: p.Progress.Pending, SignedIn: signedIn, NextAt: gameDay(time.Now()).AddDate(0, 0, 1).Unix()}
	if p.Progress.Pending && len(p.Progress.Outcomes) > 0 {
		v.Outcome = &p.Progress.Outcomes[len(p.Progress.Outcomes)-1]
	}
	if !p.Finished && !p.Progress.Pending {
		reel := g.Reels[p.Progress.Current]
		v.ReelID = reel.ID
		v.ImageURL = "https://image.tmdb.org/t/p/w780" + reel.ReelFrames[len(p.Progress.Guesses)].Frame.ImagePath
		for i, a := range reel.Alternatives {
			o := DailyOption{ID: a.ID, Key: string(rune('A' + i)), Title: a.Film.Title, Year: a.Film.Year}
			for _, id := range p.Progress.Guesses {
				if id == a.ID {
					o.Wrong = true
				}
			}
			v.Options = append(v.Options, o)
		}
	}
	return v
}

func shareMessage(v DailyView, baseURL string) string {
	text := fmt.Sprintf("Freeze Frame · %s\n%d / 1000 points\n", v.Date, v.Score)
	for _, outcome := range v.Outcomes {
		mark := "✗"
		if outcome.Correct {
			mark = "✓"
		}
		text += fmt.Sprintf("Reel %d: %s %d takes · %d points\n", outcome.Reel, mark, outcome.Takes, outcome.Points)
	}
	return "https://wa.me/?text=" + url.QueryEscape(text+baseURL+"/ui/freezeframe/?date="+v.Date)
}
