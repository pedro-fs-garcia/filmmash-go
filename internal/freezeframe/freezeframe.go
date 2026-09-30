package freezeframe

import (
	"encoding/json"
	"filmmash/internal/film"
	"fmt"
	"log/slog"
	"net/http"
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
	AlternativeID  int32 `json:"alternative_id"`
	FramesRevealed int16 `json:"frames_revealed"`
}

type StateCookie struct {
	StateID    uuid.UUID `json:"state_id"`
	GameID     int32     `json:"game_id"`
	ReelID     int32     `json:"reel_id"`
	ReelSeq    int16     `json:"reel_seq"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

func setStateCookie(w http.ResponseWriter, stateID uuid.UUID, gameID, reelID int32) error {
	state, err := json.Marshal(StateCookie{
		StateID:    stateID,
		GameID:     gameID,
		ReelID:     reelID,
		LastSeenAt: time.Now(),
	})
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "freezeframe_state",
		Value:    string(state),
		Path:     "/ui/freeze_frame",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}
