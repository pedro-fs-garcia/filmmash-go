package freezeframe

import (
	"bytes"
	"encoding/json"
	"errors"
	"filmmash/internal/auth"
	"filmmash/internal/view"
	"net/http"
	"net/url"
	"strconv"
)

func (h *Handler) renderDaily(w http.ResponseWriter, r *http.Request, v DailyView) {
	if v.Finished {
		v.ShareURL = shareMessage(v, h.baseURL)
	}
	var buf bytes.Buffer
	if err := view.FreezeFrameDaily.ExecuteTemplate(&buf, "base", v); err != nil {
		h.logger.Error("render daily game", "error", err)
		http.Error(w, "Could not display the game. Please try again.", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
func (h *Handler) DailyGame(w http.ResponseWriter, r *http.Request) {
	h.dailyRequest(w, r, false, false)
}
func (h *Handler) DailyAnswer(w http.ResponseWriter, r *http.Request) {
	h.dailyRequest(w, r, true, false)
}
func (h *Handler) DailyNext(w http.ResponseWriter, r *http.Request) {
	h.dailyRequest(w, r, false, true)
}
func (h *Handler) dailyRequest(w http.ResponseWriter, r *http.Request, answer, next bool) {
	g, err := h.dailyGame(r.Context(), r)
	if err != nil {
		h.logger.Warn("daily game unavailable", "error", err)
		message := "The projector is resting. Today's game is not ready yet. Please check back soon."
		if r.FormValue("date") != "" {
			message = "This day is unavailable. Sign in to access past games, or return to today's game."
		}
		h.renderDaily(w, r, DailyView{Error: message})
		return
	}
	key, user, err := h.player(w, r)
	if err != nil {
		http.Error(w, "Could not resume your game. Please try again.", 500)
		return
	}
	var change func(*Play) error
	if answer {
		var reelID, alternativeID int32
		var take int
		if r.Header.Get("Content-Type") == "application/json" {
			var body struct {
				ReelID        int32 `json:"reel_id"`
				AlternativeID int32 `json:"alternative_id"`
				Take          int   `json:"take"`
			}
			if err = json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Invalid answer", 400)
				return
			}
			reelID = body.ReelID
			alternativeID = body.AlternativeID
			take = body.Take
		} else {
			rid, _ := strconv.ParseInt(r.FormValue("reel_id"), 10, 32)
			aid, _ := strconv.ParseInt(r.FormValue("alternative_id"), 10, 32)
			reelID = int32(rid)
			alternativeID = int32(aid)
			take, _ = strconv.Atoi(r.FormValue("take"))
		}
		change = func(p *Play) error { return applyTake(p, g, reelID, alternativeID, take) }
	}
	if next {
		change = func(p *Play) error {
			if p.Finished || !p.Progress.Pending {
				return errStaleTake
			}
			p.Progress.Current++
			p.Progress.Guesses = nil
			p.Progress.Pending = false
			return nil
		}
	}
	p, err := h.withPlay(r.Context(), g, key, user, change)
	if err != nil {
		if errors.Is(err, errStaleTake) {
			p, err = h.withPlay(r.Context(), g, key, user, nil)
		}
		if err != nil {
			h.logger.Error("daily play failed", "error", err)
			http.Error(w, "Could not save your take. Refresh to resume.", 500)
			return
		}
	}
	h.renderDaily(w, r, dailyView(g, p, user != nil))
}

func (h *Handler) requireAccount(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/auth/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
	}
	return session, ok
}
