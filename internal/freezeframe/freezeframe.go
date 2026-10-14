package freezeframe

import (
	"context"
	"encoding/json"
	"errors"
	"filmmash/internal/film"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type GameCache struct {
	logger     *slog.Logger
	CachedGame *Game
	UpdatedAt  time.Time
}

func (gc *GameCache) Set(g *Game) {
	gc.CachedGame = g
	gc.UpdatedAt = time.Now()
	gc.logger.Info(fmt.Sprintf("new cached game: (game_id: %d)", g.ID))
}

func (gc *GameCache) Read() *Game {
	return gc.CachedGame
}

func (gc *GameCache) ValidAt() time.Time {
	return gc.CachedGame.ValidAt
}

type Game struct {
	ID      int32     `json:"id"`
	ValidAt time.Time `json:"valid_at"`
	Reels   []Reel    `json:"reels"`
}

type Reel struct {
	ID  int32 `json:"id"`
	Seq int16 `json:"seq"`

	Film         film.Film     `json:"film"`
	ReelFrames   []ReelFrame   `json:"reel_frames"`
	Alternatives []Alternative `json:"alternatives"`
}

func (r *Reel) IsRightAlternative(alternativeSeq int16) bool {
	return r.Film.Id == r.Alternatives[alternativeSeq-1].Film.Id
}

type Frame struct {
	ID        int32  `json:"id"`
	FilmID    int32  `json:"film_id"`
	ImagePath string `json:"image_path"`
}

type ReelFrame struct {
	ID         int32 `json:"id"`
	Seq        int16 `json:"seq"`
	Difficulty int16 `json:"difficulty"`
	Frame      Frame `json:"frame"`
}

type Alternative struct {
	ID   int32     `json:"id"`
	Seq  int16     `json:"seq"`
	Film film.Film `json:"film"`
}

type Answer struct {
	ID                int32     `json:"id"`
	ReelID            int32     `json:"reel_id"`
	ReelAlternativeID int32     `json:"reel_alternative_id"`
	UserID            uuid.UUID `json:"user_id"`
	FramesRevealed    int16     `json:"frames_revealed"`
	CreatedAt         time.Time `json:"created_at"`
}

type Round struct {
	GameID       int32         `json:"game_id"`
	ValidAt      time.Time     `json:"valid_at"`
	TotalReels   int           `json:"total_reels"`
	ReelID       int32         `json:"reel_id"`
	ReelSeq      int16         `json:"reel_seq"`
	NextReelID   int32         `json:"next_reel_id"`
	ReelFrames   []ReelFrame   `json:"reel_frames"`
	Alternatives []Alternative `json:"alternatives"`
}

type Submission struct {
	ReelID         int32 `json:"reel_id"`
	ReelSeq        int16 `json:"reel_seq"`
	AlternativeID  int32 `json:"alternative_id"`
	AlternativeSeq int16 `json:"alternative_seq"`
	FramesRevealed int16 `json:"frames_revealed"`
}

type StateCookie struct {
	StateID    uuid.UUID `json:"state_id"`
	GameID     int32     `json:"game_id"`
	ReelID     int32     `json:"reel_id"`
	ReelSeq    int16     `json:"reel_seq"`
	Finished   bool      `json:"finished"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

func GetStateCookie(ctx context.Context, r *http.Request) (StateCookie, error) {
	cookie, err := r.Cookie("freezeframe_state")
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		return StateCookie{}, err
	}

	state := StateCookie{}
	if cookie != nil {
		err = json.Unmarshal([]byte(cookie.Value), &state)
		if err != nil {
			return StateCookie{}, err
		}
	}
	return state, nil
}

func GetCookie[T StateCookie | ScoreCookie](ctx context.Context, r *http.Request) (T, error) {
	var t T
	var cookieName string

	switch any(t).(type) {
	case StateCookie:
		cookieName = "freezeframe_state"
	case ScoreCookie:
		cookieName = "freezeframe_score"
	}

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return t, err
	}

	if cookie != nil {
		payload, decodeErr := url.QueryUnescape(cookie.Value)
		if decodeErr != nil {
			return t, decodeErr
		}
		err = json.Unmarshal([]byte(payload), &t)
		if err != nil {
			return t, err
		}
	}
	return t, nil
}

func setStateCookie(w http.ResponseWriter, stateID uuid.UUID, gameID, reelID int32, reelSeq int16, finished bool) error {
	state, err := json.Marshal(StateCookie{
		StateID:    stateID,
		GameID:     gameID,
		ReelID:     reelID,
		ReelSeq:    reelSeq,
		Finished:   finished,
		LastSeenAt: time.Now(),
	})
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "freezeframe_state",
		Value:    url.QueryEscape(string(state)),
		Path:     "/ui/freezeframe",
		MaxAge:   24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

type ScoreCookie struct {
	GameID       int32     `json:"game_id"`
	CurrentScore int16     `json:"current_score"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

func SetScoreCookie(w http.ResponseWriter, gameID int32, score int16) error {
	payload, err := json.Marshal(ScoreCookie{
		GameID:       gameID,
		CurrentScore: score,
		LastSeenAt:   time.Now(),
	})
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "freezeframe_score",
		Value:    url.QueryEscape(string(payload)),
		Path:     "/ui/freezeframe",
		MaxAge:   24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}
