# Backlog

Feature backlog. Stories are ordered by dependency within each epic; sizes are rough (S < half a day, M ~ a day, L > a day).

---

## Epic 1 — IMDB link on the film page and modal

**Goal:** the film detail page (`/ui/film/{id}`) and the film modal show a link to the film's IMDB title page (`https://www.imdb.com/title/{imdb_id}/`).

**Context:** the `films` table has no IMDB id — its primary key is the TMDB id (`migrations/20260530214103_init_schema.sql`). The IMDB `tt…` id must come from TMDB and be stored, since building the link at render time would cost a TMDB call per view.

### 1.1 — Store `imdb_id` on films (M)

Add the column and thread it through the data layer.

- [ ] New goose migration: `ALTER TABLE films ADD COLUMN imdb_id TEXT` (nullable — TMDB can return null, and existing rows won't have it until backfilled). Optional `CHECK (imdb_id ~ '^tt\d+$')`.
- [ ] Update `InsertFilm` in `internal/database/queries/films.sql` and every film `SELECT` used by `GetFilm`/list queries to include `imdb_id`; regenerate sqlc (`sqlc.yaml`).
- [ ] Add `ImdbId string` to `film.Film` (`internal/film/film.go`) and map it in `internal/film/repository.go`.

**Acceptance criteria:** migration applies cleanly on a fresh DB and on the existing prod schema; `film.Service.GetFilm` returns the stored id. Tests run against the PG18 container (port 5440 — migrations use `uuidv7()`).

### 1.2 — Fetch `imdb_id` from TMDB on insert (S)

- [ ] Add `ImdbId string \`json:"imdb_id"\`` to `FilmDetailResponse` and `Movie` in `internal/tmdb` — the `/movie/{id}` details endpoint already returns it, so `GetFilm` (used by the admin "add from TMDB" flow, `internal/admin/service.go`) gets it for free.
- [ ] Seed path (`cmd/seed/main.go` via `GetPopulars`/`GetTopRated`): the list endpoints do **not** return `imdb_id`. Since these already make one `GetDirector` call per movie, replace that per-movie call with `GetFilm` (details + `append_to_response=credits`), which returns director **and** `imdb_id` in a single request — no extra API traffic.

**Acceptance criteria:** adding a film through `/admin/films/search` → insert stores its `imdb_id`; seeding stores `imdb_id` for every film that has one on TMDB.

### 1.3 — Backfill `imdb_id` for existing films (M)

- [ ] One-shot backfill command (e.g. `cmd/backfill-imdb/`, or a flag on the seed cmd): select films where `imdb_id IS NULL`, call TMDB `/movie/{id}/external_ids` (or the details endpoint), update the row. Modest throttling is enough (TMDB allows ~50 req/s).
- [ ] Log films for which TMDB has no IMDB id; leave them NULL.

**Acceptance criteria:** after running against prod data, all films that have an IMDB id on TMDB have it stored; the command is idempotent and safe to re-run.

### 1.4 — Render the IMDB link (S)

- [ ] `internal/view/template/film.html` (film page) and `film_modal.html`: add an external link `https://www.imdb.com/title/{{ .ImdbId }}/` with `target="_blank" rel="noopener noreferrer"`, rendered **only when `ImdbId` is non-empty** (`{{ if .ImdbId }}`).
- [ ] In the modal, place it inside `.modal-film` near the rating; style per `docs/visual-identity.md`.

**Acceptance criteria:** film page and modal show a working IMDB link for films with an id, and nothing (no dead link) for films without one.

---

## Epic 2 — Freeze Frame (daily frame-guessing game)

**Goal:** *Freeze Frame* — a daily game where the player identifies films from single frames, scores up to 1000 points, and can share the result on WhatsApp.

**Context:** requirements only — no implementation decisions here.

### Vocabulary

- **Freeze Frame** — the name of the feature, as shown to players.
- **Frame** — a still image taken from a film.
- **Take** — one attempt at a Reel: the player picks an option. Up to 4 Takes per Reel.
- **Reel** — one film to guess: up to 4 Frames of that film, one set of 5 options, up to 4 Takes.
- **Game** — 5 consecutive Reels. One Game per player per day.

### 2.1 — Functional requirements

**Game structure**

- [ ] A Game is composed of exactly 5 Reels, played in sequence.
- [ ] Each Reel presents one Frame of a film and 5 film options, exactly one of which is correct.
- [ ] The 5 options stay the same for the whole Reel — they do not change between Takes.
- [ ] Each Reel uses a different film; no film repeats within a Game.

**Playing a Reel**

- [ ] The player picks one of the 5 options — that is one Take.
- [ ] Correct answer → the Reel ends and the next Reel starts (new film, new Frame, new set of 5 options).
- [ ] Wrong answer → a different Frame **of the same film** is shown, with the same 5 options. The option already used is marked as wrong and can no longer be chosen.
- [ ] A Reel allows at most 4 Takes. After the 4th wrong answer the Reel ends with 0 points and the Game moves on to the next Reel.
- [ ] A Reel cannot be skipped or abandoned — the player either guesses correctly or exhausts the 4 Takes. There is no way to reach the next Reel without answering.
- [ ] After a Reel ends the player is told which film it was, whether they got it right, and how many points they earned.
- [ ] Frames within a Reel are shown in a fixed difficulty order: the first Frame is the hardest, each subsequent Frame is easier.

**Scoring**

- [ ] A Reel is worth 200 points on the 1st Take, 150 on the 2nd, 100 on the 3rd, 50 on the 4th, and 0 if all 4 Takes fail (−50 per failed Take).
- [ ] A Game is worth at most 1000 points (5 Reels × 200).
- [ ] The running total is visible during the Game, and the final score is shown at the end.

**Access and the one-Game-a-day rule**

- [ ] No account is required to play; a visitor can start the daily Game straight away.
- [ ] A player gets one Game per day. For anonymous players the Game is identified as "played" via a cookie set on the device; for logged-in players it is tied to the account, so it holds across devices and browsers.
- [ ] A player who has already played that day sees their result for the day and the time until the next Game, instead of a new Game.
- [ ] An interrupted Game (page closed, browser reloaded) can be resumed at the Reel and Take where it stopped — it does not count as finished and does not restart from Reel 1.
- [ ] All players get the same 5 films on a given day, so scores are comparable.
- [ ] A player who logs in after playing anonymously on the same device keeps that day's result — the Game is attributed to the account and does not become playable again.

**Logged-in players**

- [ ] Only logged-in players get the Game archive: they can see past Games and play days they missed. Anonymous players only ever get the current day.
- [ ] A past Game can be played once, follows the same rules, and its score counts in the player's history. Days already played cannot be replayed for a better score.
- [ ] Logged-in players get a score history (per-day score and per-Reel outcome), a streak count of consecutive days played, and a leaderboard.
- [ ] Streaks are based on the calendar day a Game was played for — an archived day played late does not repair a broken streak.
- [ ] The leaderboard ranks logged-in players only, and covers at least the current day; overall/period ranking is desirable.

**Sharing**

- [ ] At the end of a Game the player can share the result on WhatsApp with one tap.
- [ ] The shared message leads with the name *Freeze Frame* and contains the date, the final score, a per-Reel summary that does **not** reveal the answers, and a link to the Game.
- [ ] A player arriving via a shared link lands on the day's Game and can play it (subject to the one-a-day rule).

**Content**

- [ ] Frames are sourced from TMDB (film stills/backdrops) for films already in the catalogue.
- [ ] Every film used in a Reel has at least 4 Frames available, ordered from hardest to easiest.
- [ ] The admin team curates each film's Frames: which Frames are usable, and their difficulty order. A film is only eligible for a Reel once it has been curated.
- [ ] Admins can preview a film's Frames in the order players will see them, reorder them, and discard unusable ones.
- [ ] Frames must not contain the film's title, credits, or obvious on-screen text giving away the answer — this is part of what admin curation rejects.
- [ ] The 4 wrong options are plausible — same broad era/genre where possible, never the same film under another title.
- [ ] There must always be enough curated films to fill upcoming daily Games; admins can see how many days are covered.
- [ ] TMDB attribution requirements for the Frames are respected wherever they are shown.

### 2.2 — Non-functional requirements

- [ ] **Fairness:** the correct option's position among the 5 is random per Reel; the day's films and Frames are the same for everyone.
- [ ] **Anti-cheating (best effort):** the correct answer is not discoverable from the page before the player answers. For anonymous players, cookie-based limiting is understood to be bypassable (cleared cookies, private browsing); this is accepted, not defended against. Scores that appear on the leaderboard come from logged-in accounts, where the limit is enforced server-side.
- [ ] **Privacy:** anonymous play requires no personal data. The play cookie is strictly functional and holds no identifying information; no third-party tracking is added by this feature. Leaderboard entries show only what the player already exposes elsewhere on the site.
- [ ] **Day boundary:** "a day" is a single, explicitly defined timezone for all players, and the countdown to the next Game reflects it.
- [ ] **Performance:** a Frame is visible within ~1 s on a normal mobile connection; answering feels instant (< 500 ms to the next Frame or Reel).
- [ ] **Mobile first:** the Game is fully playable on a phone in portrait — Frame legible without zooming, options tappable one-handed. Sharing is the primary end-of-Game action.
- [ ] **Accessibility:** options are keyboard-reachable and screen-reader labelled; correct/wrong is never signalled by colour alone.
- [ ] **Availability:** if the Game cannot be loaded, the player gets a clear message and the rest of the site keeps working. A failed load must never consume the player's daily attempt.
- [ ] **Footprint:** the feature must fit the existing small VPS — Frame images stay small enough to serve without a heavier hosting setup.

**Acceptance criteria:** a first-time anonymous visitor plays 5 Reels under the rules above, sees a score between 0 and 1000, shares it on WhatsApp, and on reload sees "already played today" with a countdown; a second visitor the same day gets the same 5 films; a logged-in player sees their history, streak and the leaderboard, can play a missed day from the archive, and cannot replay a day they already played on any device.
