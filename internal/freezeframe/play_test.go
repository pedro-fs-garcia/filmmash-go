package freezeframe

import (
	"bytes"
	"encoding/json"
	"filmmash/internal/film"
	"filmmash/internal/view"
	"net/url"
	"strings"
	"testing"
	"time"
)

func rulesGame() Game {
	g := Game{ID: 1, ValidAt: gameDay(time.Now())}
	for i := 0; i < 5; i++ {
		r := Reel{ID: int32(i + 1), Seq: int16(i + 1), Film: film.Film{Id: 100 + i, Title: "Secret film"}}
		for j := 0; j < 5; j++ {
			id := 200 + j
			if j == 4 {
				id = 100 + i
			}
			r.Alternatives = append(r.Alternatives, Alternative{ID: int32(i*10 + j + 1), Seq: int16(j + 1), Film: film.Film{Id: id, Title: "Option"}})
		}
		for j := 0; j < 4; j++ {
			r.ReelFrames = append(r.ReelFrames, ReelFrame{Frame: Frame{ImagePath: "/frame" + string(rune('a'+j)) + ".jpg"}})
		}
		g.Reels = append(g.Reels, r)
	}
	return g
}
func TestTakeScoresAndDisabledChoices(t *testing.T) {
	g := rulesGame()
	for take, want := range []int{200, 150, 100, 50} {
		p := Play{}
		for i := 0; i < take; i++ {
			if err := applyTake(&p, g, 1, int32(i+1), i+1); err != nil {
				t.Fatal(err)
			}
			if p.Progress.Pending {
				t.Fatal("wrong answer ended reel early")
			}
		}
		if err := applyTake(&p, g, 1, 5, take+1); err != nil {
			t.Fatal(err)
		}
		if p.Score != want || !p.Progress.Pending {
			t.Fatalf("take %d: got %d, want %d", take+1, p.Score, want)
		}
	}
	p := Play{}
	if err := applyTake(&p, g, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := applyTake(&p, g, 1, 1, 2); err == nil {
		t.Fatal("repeated wrong option accepted")
	}
	if err := applyTake(&p, g, 1, 2, 1); err == nil {
		t.Fatal("stale take accepted")
	}
	v := dailyView(g, p, false)
	if v.Take != 2 || !v.Options[0].Wrong || v.ImageURL != "https://image.tmdb.org/t/p/w780/frameb.jpg" {
		t.Fatal("wrong answer did not reveal next frame and disable choice")
	}
}
func TestFourWrongTakesAndFinalReel(t *testing.T) {
	g := rulesGame()
	p := Play{}
	for i := 1; i <= 4; i++ {
		if err := applyTake(&p, g, 1, int32(i), i); err != nil {
			t.Fatal(err)
		}
	}
	if p.Score != 0 || !p.Progress.Pending || p.Finished || p.Progress.Outcomes[0].Correct {
		t.Fatal("four failures must end only this reel")
	}
	if err := applyTake(&p, g, 1, 5, 5); err == nil {
		t.Fatal("fifth take accepted")
	}
	p = Play{Progress: PlayProgress{Current: 4}, Score: 800}
	if err := applyTake(&p, g, 5, 45, 1); err != nil {
		t.Fatal(err)
	}
	if !p.Finished || p.Score != 1000 {
		t.Fatal("final take not scored")
	}
	if err := applyTake(&p, g, 5, 45, 1); err == nil {
		t.Fatal("completed game replayed")
	}
}
func TestResumeAndNoAnswerLeak(t *testing.T) {
	g := rulesGame()
	p := Play{}
	_ = applyTake(&p, g, 1, 1, 1)
	data, _ := json.Marshal(p.Progress)
	var resumed Play
	_ = json.Unmarshal(data, &resumed.Progress)
	if dailyView(g, resumed, false).Take != 2 {
		t.Fatal("take not resumed")
	}
	var b bytes.Buffer
	if err := view.FreezeFrameDaily.ExecuteTemplate(&b, "base", dailyView(g, Play{}, false)); err != nil {
		t.Fatal(err)
	}
	html := b.String()
	if strings.Contains(html, "Secret film") || strings.Contains(html, "frameb.jpg") || strings.Contains(html, "Reveal Next Frame") {
		t.Fatal("unrevealed answer/frame leaked")
	}
	if strings.Count(html, `name="alternative_id"`) != 5 {
		t.Fatal("must show five options")
	}
	for _, v := range []DailyView{dailyView(g, p, false), dailyView(g, Play{Finished: true, Score: 1000, Progress: PlayProgress{Outcomes: []Outcome{{Reel: 1, Film: "Secret film", Correct: true, Takes: 1, Points: 200}}}}, true), {Error: "Unavailable"}} {
		b.Reset()
		if err := view.FreezeFrameDaily.ExecuteTemplate(&b, "base", v); err != nil {
			t.Fatal(err)
		}
	}
}
func TestDayAndSharing(t *testing.T) {
	at := time.Date(2026, 10, 13, 23, 30, 0, 0, time.FixedZone("west", -3*3600))
	if gameDay(at).Format("2006-01-02") != "2026-10-14" {
		t.Fatal("UTC boundary incorrect")
	}
	v := DailyView{Date: "2026-10-13", Score: 150, Outcomes: []Outcome{{Reel: 1, Film: "Never share this answer", Correct: true, Takes: 2, Points: 150}}}
	u, _ := url.Parse(shareMessage(v, "https://example.com"))
	message := u.Query().Get("text")
	if !strings.HasPrefix(message, "Freeze Frame") || strings.Contains(message, "Never share") || !strings.Contains(message, "2026-10-13") || !strings.Contains(message, "150 / 1000") {
		t.Fatal("invalid share payload")
	}
}
