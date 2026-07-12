# FilmMash — Feature Backlog

Stories follow the model in [backlog.md](backlog.md). Ratings referenced below are Elo points (start `1400`, current fixed `K = 20` — see `internal/film/film.go`).

| ID | Story | Depends on |
|---|---|---|
| US-01 | Seed popular and top-rated films from TMDB | — |
| US-02 | Rating-constrained duel matchmaking | — |
| US-03 | Popularity-weighted duel selection | US-01 |
| US-04 | Admin adds films after seeding | — |
| US-05 | Selection boost for newly added films | US-02/US-03 (composes with) |
| US-06 | Provisional ratings for new films | US-05 (shares `duel_count`) |

---

# 📑 US-01: Seed popular and top-rated films from TMDB

## User Story
**As** the platform operator
**I want** the seeder to import both TMDB "Top Rated" and "Popular" lists
**So that** the catalog mixes acclaimed classics with films people currently care about

## Business Rules

1. `cmd/seed` fetches both `/movie/top_rated` and `/movie/popular`. The client already implements `GetTopRated` and `GetPopulars` (`internal/tmdb/services.go`); today only `GetTopRated` is called.
2. Page depth is configurable per list (defaults: 25 pages top-rated — current behavior — and 10 pages popular).
3. The TMDB `popularity` value is parsed (new field on `tmdb.Movie`) and persisted in a new `films.popularity` column (default `0`). US-03 depends on it.
4. Films whose director cannot be resolved from `/movie/{id}/credits` are skipped (`films.director_id` is `NOT NULL`).
5. Seeding is idempotent: a film present in both lists (or already in the DB) is inserted once. Re-runs may refresh `popularity`, but must never touch `rating`, which starts at the default `1400` and belongs to the Elo system afterwards.

## Entities

- **Film** — new column `popularity FLOAT NOT NULL DEFAULT 0` (migration required)
- **Director**
- **TMDB Movie** — new parsed field `popularity`

## Acceptance Criteria

### Case 1: Seeding an empty database

**Given that** the `films` and `directors` tables are empty
**And** the TMDB API is reachable with a valid token
**When** the seed command runs
**Then** films from both the Top Rated and Popular lists exist in the database, each with its director, its TMDB `popularity` stored, and `rating = 1400`

### Case 2: Re-running the seeder does not disturb ratings

**Given that** the database is already seeded
**And** film X has been voted on and its rating is no longer `1400`
**When** the seed command runs again
**Then** no duplicate films or directors are created
**And** film X's `rating` is unchanged (only its `popularity` may be refreshed)

### Case 3: A film appears in both TMDB lists

**Given that** a movie is present in both the Top Rated and the Popular results
**When** the seed command runs
**Then** the film is stored exactly once

---

# 📑 US-02: Rating-constrained duel matchmaking

## User Story
**As** a voter
**I want** duels to pair films of similar rating
**So that** every choice is a genuine contest instead of an obvious mismatch

## Business Rules

1. A duel's films must satisfy `|rating_a − rating_b| ≤ window`. Default window: `150` points, configurable via env (`DUEL_RATING_WINDOW`).
2. The constraint applies to both selection paths: the random duel (`SelectRandomFilms`, today `ORDER BY RANDOM() LIMIT 2`) and the winner-stays challenger (`ComposeDuel`, today any random film ≠ winner).
3. Fallback: if no candidate exists within the window, the window doubles progressively until a candidate is found. Duel creation never fails while the catalog has at least 2 films.
4. Selection stays a single SQL query inside the existing transaction pattern (`TxManager` / sqlc); the Elo update itself is unchanged.

## Entities

- **Duel**
- **Film** — `rating`
- **Config** — new env var `DUEL_RATING_WINDOW`

## Acceptance Criteria

### Case 1: Random duel respects the window

**Given that** the catalog contains films across a wide rating range
**When** a new random duel is created
**Then** the two selected films' ratings differ by at most the configured window

### Case 2: Winner-stays challenger respects the window

**Given that** a user has just voted and the winner has rating R
**When** the next duel is composed for that winner
**Then** the challenger's rating is within the window of R

### Case 3: Sparse catalog falls back gracefully

**Given that** no film exists within the window of the selected film
**And** the catalog has at least 2 films
**When** a duel is created
**Then** the window is widened progressively until a challenger is found
**And** duel creation succeeds

---

# 📑 US-03: Popularity-weighted duel selection

## User Story
**As** the product owner
**I want** popular films to show up in duels more often
**So that** voters mostly judge films they recognize, keeping duels engaging

## Business Rules

1. Candidate selection is weighted by `films.popularity` (from US-01): selection weight = `1 + popularity × strength`. The `+1` baseline guarantees every film remains selectable.
2. `strength` is configurable via env (`POPULARITY_WEIGHT`, default `1.0`); `0` restores uniform random selection.
3. Composes with US-02: first filter candidates by the rating window, then take a weighted random pick among them (e.g. the `ORDER BY -LN(RANDOM()) / weight` technique, in SQL).
4. Applies to both selection paths (random duel and winner-stays challenger).

## Entities

- **Film** — `popularity` (from US-01)
- **Duel**
- **Config** — new env var `POPULARITY_WEIGHT`

## Acceptance Criteria

### Case 1: Popular films appear more often

**Given that** two films have equal ratings but film A's popularity is far higher than film B's
**When** a large number of duels is created
**Then** film A appears in significantly more duels than film B

### Case 2: Unpopular films still appear

**Given that** a film has `popularity = 0`
**When** duels are created over time
**Then** the film is still eventually selected (its weight is the baseline, never zero)

### Case 3: Weighting can be disabled

**Given that** `POPULARITY_WEIGHT` is set to `0`
**When** duels are created
**Then** selection is uniformly random among the rating-window candidates

---

# 📑 US-04: Admin adds films after seeding

## User Story
**As** an admin
**I want** to search TMDB and add a film from the admin dashboard
**So that** the catalog can grow continuously without re-running the seeder

## Business Rules

1. The feature lives under `/admin` and is protected by the existing `RequiresRole("admin")` middleware.
2. The admin searches TMDB by title (new client method for `/search/movie`); results show title, release year, and poster. Picking one makes the server fetch the director via `/movie/{id}/credits` and insert film + director, reusing the seeder's persistence rules (US-01), including `popularity`.
3. A film that already exists (same TMDB id) is reported as already in the catalog; nothing is modified.
4. A film whose director cannot be resolved cannot be added (`director_id` is `NOT NULL`).
5. The new film starts at `rating = 1400`, records `created_at = NOW()`, and is subject to the new-film treatment of US-05/US-06 once those ship.
6. Schema validation applies: title ≤ 255 chars, release year within 1800–2100.
7. The flow is server-rendered HTML + HTMX fragments, consistent with the existing `admin_dashboard`/`admin_users` templates and the visual identity ([visual-identity.md](visual-identity.md)).

## Entities

- **Film**
- **Director**
- **User / Session** — `roles` (admin gate)
- **TMDB Movie** — search results

## Acceptance Criteria

### Case 1: Admin adds a film successfully

**Given that** a logged-in user has the `admin` role
**And** the film is not yet in the catalog
**When** the admin searches TMDB, picks a result, and confirms
**Then** the film and its director are stored with `rating = 1400` and its TMDB popularity
**And** the film becomes eligible for duels immediately

### Case 2: Duplicate film is rejected gracefully

**Given that** the chosen film already exists in the catalog
**When** the admin confirms the addition
**Then** the UI reports it is already in the catalog
**And** the existing row (including its rating) is unchanged

### Case 3: Non-admin cannot access the feature

**Given that** a user without the `admin` role is logged in (or no user is logged in)
**When** they request the add-film page or endpoint
**Then** the request is denied by the role middleware

### Case 4: Search with no results

**Given that** the admin searches for a title TMDB does not know
**When** the search returns
**Then** an empty-state message is shown and nothing can be submitted

---

# 📑 US-05: Selection boost for newly added films

## User Story
**As** the product owner
**I want** recently added films to enter duels more frequently
**So that** they gather votes quickly and find their real place in the ranking

## Business Rules

1. A film counts as **new** until it has played `N` duels (default `10`, configurable via `NEW_FILM_DUEL_THRESHOLD`).
2. Duels played are tracked in a new `films.duel_count` column, incremented for both films inside the existing vote transaction (US-06 reuses this column). Counting votes on the fly is the fallback, but the column keeps selection queries cheap.
3. While new, a film's selection weight is multiplied by a boost factor (default `×5`, configurable via `NEW_FILM_BOOST`).
4. Composition order across US-02/US-03/US-05: filter candidates by rating window → weight = `(1 + popularity × strength) × (boost if new)` → weighted random pick.
5. The boost expires automatically once `duel_count ≥ N`; no scheduled job is needed.

## Entities

- **Film** — new column `duel_count INT NOT NULL DEFAULT 0` (migration required)
- **Vote** — its transaction increments `duel_count`
- **Duel**
- **Config** — new env vars `NEW_FILM_DUEL_THRESHOLD`, `NEW_FILM_BOOST`

## Acceptance Criteria

### Case 1: New film is boosted

**Given that** a film has `duel_count` below the threshold
**When** duel candidates are weighted
**Then** that film's selection weight is multiplied by the configured boost

### Case 2: Boost expires

**Given that** a film's `duel_count` has reached the threshold
**When** the next duel selection happens
**Then** the film competes with its normal (unboosted) weight

### Case 3: Duel count is maintained by voting

**Given that** a duel between films A and B exists
**When** a vote on that duel is registered
**Then** `duel_count` is incremented by 1 for both A and B within the same transaction as the rating update

---

# 📑 US-06: Provisional ratings — higher volatility for new films

## User Story
**As** the product owner
**I want** films with few duels to have their rating move in bigger steps
**So that** a film converges to its true rank after a handful of votes instead of hundreds

## Business Rules

1. The fixed `K = 20` (`film.CalculateRatings`) is replaced by a per-film K-factor derived from `duel_count` (US-05): `K = 40` while `duel_count < 10`, `K = 30` while `< 20`, `K = 20` otherwise (thresholds configurable).
2. Winner and loser may have different K values, so deltas are computed independently: `winner' = winner + K_w × (1 − E_w)` and `loser' = loser + K_l × (0 − E_l)`. `CalculateRatings` changes signature to receive each film's K (or duel count).
3. With asymmetric K the total rating mass is intentionally not conserved on mixed pairings — the standard trade-off of provisional ratings; established films are still protected by their own lower K.
4. The K lookup uses the same rows already locked `FOR UPDATE` in the vote transaction (`GetDuelRatingsForUpdate` gains `duel_count`), so no extra query or race is introduced.
5. `votes.winner_rating_after` / `loser_rating_after` keep their meaning and continue to be recorded.

## Entities

- **Film** — `rating`, `duel_count` (from US-05)
- **Vote**

## Acceptance Criteria

### Case 1: New film swings harder than an established one

**Given that** film A has `duel_count = 2` (K = 40) and film B has `duel_count = 50` (K = 20)
**And** both have rating 1400
**When** A beats B
**Then** A gains 20 points (40 × 0.5) and B loses 10 points (20 × 0.5)

### Case 2: Established pair keeps today's behavior

**Given that** both films have `duel_count` ≥ 20 and equal ratings
**When** a vote is registered
**Then** the winner gains 10 and the loser loses 10 (K = 20 on both sides), exactly as the current implementation

### Case 3: Volatility decreases as a film plays

**Given that** a film crosses a K threshold (e.g. its `duel_count` reaches 10)
**When** its next vote is registered
**Then** the lower K tier is applied to its delta
