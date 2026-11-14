# Freeze Frame operations

The daily calendar is **UTC**, resetting at 00:00 UTC for every player. Games have five distinct films, four curated frames per reel and five alternatives. Frames appear in curator order, hardest first. Wrong takes reveal the next frame; correct answers or four failures end the reel. Scores are 200, 150, 100, 50 or zero.

## Prepare content

Visit `/admin/freezeframe` with an admin account. Select a catalogue film and import its TMDB backdrops. Assign exactly four usable images positions 1–4, confirm that no titles, credits or answer-revealing text are present, preview the selected order, and save. Unselected images are excluded. A film is eligible only after all four positions are approved.

Schedule a date using five different eligible films. The scheduler snapshots the frames, selects four distractors from nearby release years, excludes identical known IMDB identities, and shuffles the five options once. Every player gets the same scheduled content and option order. Editing curation later does not alter scheduled reels. Coverage shows the number of scheduled dates from today onward; keep several days prepared.

Legacy games with four options and five frames are not automatically converted or marked curated. The calendar migration marks them inactive while preserving their rows and references. Admins can schedule a compliant replacement for those dates; a partial unique index enforces one active game per date. Schedule new compliant games rather than inventing approval for old placeholder images. Legacy score/state cookies are ignored by the new flow.

## Progress and accounts

`freeze_frame_plays` stores current reel/take, used choices, outcomes and score. A strictly functional HTTP-only cookie contains a random device token; the database stores its hash. No personal data, IP address, tracking script or third-party identifier is collected for anonymous play. Failed game loads create no play record. Account plays have a unique `(game_id, user_id)` constraint, and answers are processed inside a row-locked transaction. Duplicate requests cannot add points or skip takes.

On an authenticated game/history visit, this device's anonymous progress is attributed to the account. A completed account result is never replaced. The more advanced unfinished progress is retained. Another authenticated device resumes the account row. Logging out returns to the device's anonymous row; clearing the anonymous cookie remains an accepted way to bypass the device limit.

History and archived games require an account. Completed days are read-only. A streak counts only games completed on their scheduled UTC date; playing an old date late does not repair it. The leaderboard ranks authenticated completions and uses stable generated aliases instead of exposing internal account ids or private profile information. WhatsApp sharing contains the date, score, outcomes without film names, and the dated game URL.

The game admin JSON endpoints `/api/freezeframe` and `/api/game` require the admin role so game answers are not public. JSON creation also requires approved frames in curated order. `/ui/freeze_frame` redirects to `/ui/freezeframe/`.

## IMDB data

Apply migrations before deploying the application or seed binary. Film inserts store nullable `imdb_id`; the film page and modal link to IMDB only when it is present. TMDB list ingestion uses one details/credits request per film instead of a separate director request.

Run `go run ./cmd/backfill-imdb` in an environment configured for the target database and TMDB. It fetches only rows with NULL ids, defaults to 100 ms between calls, logs unavailable ids, and leaves them NULL. Successful rows are skipped on later runs; transient failures return a nonzero exit status and can be retried. Use `-delay` to adjust the interval. Credentials come from the existing config and are not printed.

## Verification

Use a dedicated PostgreSQL 18 test database with current migrations. Set `DATABASE_URL` explicitly (it takes precedence over `.env.test`) and run `go test -p 1 ./...`. Packages share database fixtures, so parallel package execution can interfere with the existing duel tests. Integration tests exercise all score levels, concurrent duplicate submissions, resuming takes, guest adoption, another authenticated device, archives, streaks, leaderboard, admin authorization, import/curation and duplicate-date scheduling.

Only the current frame is sent to the player. Images load from TMDB at `w780`, with dimensions reserved to avoid layout shifts; the application does not store large image files on the VPS. The sub-second frame-load and 500 ms answer targets still need measurement on a representative mobile connection. TMDB attribution is shown on game and admin pages and in the site footer.
