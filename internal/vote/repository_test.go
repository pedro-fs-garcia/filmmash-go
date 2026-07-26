package vote_test

import (
	"context"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/testdb"
	"filmmash/internal/vote"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedFilms(t *testing.T, ctx context.Context) (film.Film, film.Film, duel.Duel) {
	t.Helper()
	filmRepo := film.NewRepository(testPool)
	duelRepo := duel.NewRepository(testPool)
	var testDirector = film.Director{Id: 543, Name: "Testing director"}
	a := film.Film{Id: 2001, Title: "Alpha", Year: 2000, Director: testDirector, ImagePath: "a.jpg"}
	b := film.Film{Id: 2002, Title: "Beta", Year: 2001, Director: testDirector, ImagePath: "b.jpg"}
	for _, f := range []*film.Film{&a, &b} {
		if err := filmRepo.InsertFilm(ctx, f); err != nil {
			t.Fatalf("seeding film %d: %v", f.Id, err)
		}
	}
	d := duel.Duel{FilmA: &a, FilmB: &b}
	if err := duelRepo.Insert(ctx, &d); err != nil {
		t.Fatalf("seeding duel: %v", err)
	}
	return a, b, d
}

func filmDuelCount(t *testing.T, ctx context.Context, filmId int) int {
	t.Helper()
	q := database.ExtractTx(ctx, testPool)
	var n int
	if err := q.QueryRow(ctx, `SELECT duel_count FROM films WHERE id = $1`, filmId).Scan(&n); err != nil {
		t.Fatalf("reading duel_count for film %d: %v", filmId, err)
	}
	return n
}

func IsValidUUID(t *testing.T, u string) bool {
	t.Helper()
	parsed, err := uuid.Parse(u)
	if err != nil {
		return false
	}
	if parsed.Version() != 7 || parsed.Variant() != uuid.RFC4122 {
		return false
	}
	return true
}

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, cleanup, err := testdb.New(ctx)
	if err != nil {
		log.Fatalf("test db setup; %v", err)
	}
	testPool = pool
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestRepository(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := vote.NewRepository(testPool)
	duelRepo := duel.NewRepository(testPool)

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txCtx := database.InjectTx(ctx, tx)
	fa, fb, d := seedFilms(t, txCtx)

	testVote := vote.Vote{
		DuelId:            d.Id,
		WinnerID:          fa.Id,
		LoserId:           fb.Id,
		WinnerRatingAfter: 1410.0,
		LoserRatingAfter:  1390.0,
	}

	t.Run("InsertVote", func(t *testing.T) {
		v := testVote
		if err := repo.InsertVote(txCtx, &v); err != nil {
			t.Fatalf("InsertVote: %v", err)
		}
		if !IsValidUUID(t, v.Id.String()) {
			t.Fatalf("got %v, want VERSION_7", v.Id.Version())
		}
	})

	t.Run("InsertVote increments both films' duel_count", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer func() {
			_ = ntx.Rollback(ctx)
		}()

		nctx := database.InjectTx(ctx, ntx)

		beforeWinner := filmDuelCount(t, nctx, fa.Id)
		beforeLoser := filmDuelCount(t, nctx, fb.Id)

		nd := duel.Duel{FilmA: &fa, FilmB: &fb}
		if err := duelRepo.Insert(nctx, &nd); err != nil {
			t.Fatalf("seeding duel: %v", err)
		}
		v := testVote
		v.DuelId = nd.Id
		if err := repo.InsertVote(nctx, &v); err != nil {
			t.Fatalf("InsertVote: %v", err)
		}

		if got := filmDuelCount(t, nctx, fa.Id); got != beforeWinner+1 {
			t.Fatalf("winner duel_count = %d, want %d", got, beforeWinner+1)
		}
		if got := filmDuelCount(t, nctx, fb.Id); got != beforeLoser+1 {
			t.Fatalf("loser duel_count = %d, want %d", got, beforeLoser+1)
		}
	})

	t.Run("InsertVote duplicate duel should fail", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer func() {
			_ = ntx.Rollback(ctx)
		}()
		nctx := database.InjectTx(ctx, ntx)

		nd := duel.Duel{FilmA: &fa, FilmB: &fb}
		if err := duelRepo.Insert(nctx, &nd); err != nil {
			t.Fatalf("seeding duel: %v", err)
		}

		first := testVote
		first.DuelId = nd.Id
		if err := repo.InsertVote(nctx, &first); err != nil {
			t.Fatalf("first InsertVote: %v", err)
		}

		dup := testVote
		dup.DuelId = nd.Id
		err = repo.InsertVote(nctx, &dup)
		if err == nil {
			t.Fatal("got no error, want ErrDuplicateEntry")
		}
		if !errors.Is(err, vote.ErrDuplicateEntry) {
			t.Fatalf("got %v, want ErrDuplicateEntry", err)
		}
	})

	t.Run("CurrentTotal", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer func() {
			_ = ntx.Rollback(ctx)
		}()
		nctx := database.InjectTx(ctx, ntx)

		before, err := repo.CurrentTotal(nctx)
		if err != nil {
			t.Fatalf("CurrentTotal: %v", err)
		}

		nd := duel.Duel{FilmA: &fa, FilmB: &fb}
		if err := duelRepo.Insert(nctx, &nd); err != nil {
			t.Fatalf("seeding duel: %v", err)
		}
		v := testVote
		v.DuelId = nd.Id
		if err := repo.InsertVote(nctx, &v); err != nil {
			t.Fatalf("InsertVote: %v", err)
		}

		after, err := repo.CurrentTotal(nctx)
		if err != nil {
			t.Fatalf("CurrentTotal: %v", err)
		}
		if after != before+1 {
			t.Fatalf("CurrentTotal after insert = %d, want %d", after, before+1)
		}
	})
}

func seedUser(t *testing.T, ctx context.Context, sub string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	q := database.ExtractTx(ctx, testPool)
	err := q.QueryRow(ctx, `INSERT INTO users (idp_sub) VALUES ($1) RETURNING id`, sub).Scan(&id)
	if err != nil {
		t.Fatalf("seeding user %q: %v", sub, err)
	}
	return id
}

func seedDuel(t *testing.T, ctx context.Context, a, b *film.Film) duel.Duel {
	t.Helper()
	d := duel.Duel{FilmA: a, FilmB: b}
	if err := duel.NewRepository(testPool).Insert(ctx, &d); err != nil {
		t.Fatalf("seeding duel: %v", err)
	}
	return d
}

func backdateVote(t *testing.T, ctx context.Context, id uuid.UUID, at time.Time) {
	t.Helper()
	q := database.ExtractTx(ctx, testPool)
	if _, err := q.Exec(ctx, `UPDATE votes SET created_at = $1 WHERE id = $2`, at, id); err != nil {
		t.Fatalf("backdating vote %s: %v", id, err)
	}
}

func TestListFilmVotes(t *testing.T) {
	ctx := context.Background()
	repo := vote.NewRepository(testPool)

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txCtx := database.InjectTx(ctx, tx)

	fa, fb, d := seedFilms(t, txCtx)

	// fa (Alpha) beats fb (Beta).
	v := vote.Vote{
		DuelId:            d.Id,
		WinnerID:          fa.Id,
		LoserId:           fb.Id,
		WinnerRatingAfter: 1410.0,
		LoserRatingAfter:  1390.0,
	}
	if err := repo.InsertVote(txCtx, &v); err != nil {
		t.Fatalf("InsertVote: %v", err)
	}

	t.Run("returns the matchup when the film is the winner", func(t *testing.T) {
		got, err := repo.ListFilmVotes(txCtx, fa.Id)
		if err != nil {
			t.Fatalf("ListFilmVotes: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d matchups, want 1", len(got))
		}
		m := got[0]
		if m.Winner.Id != fa.Id || m.Loser.Id != fb.Id {
			t.Fatalf("got winner=%d loser=%d, want winner=%d loser=%d", m.Winner.Id, m.Loser.Id, fa.Id, fb.Id)
		}
		if m.Winner.Title != "Alpha" || m.Loser.Title != "Beta" {
			t.Fatalf("got winner title %q loser title %q, want %q / %q", m.Winner.Title, m.Loser.Title, "Alpha", "Beta")
		}
		if m.Winner.Year != 2000 || m.Loser.Year != 2001 {
			t.Fatalf("got winner year %d loser year %d, want 2000 / 2001", m.Winner.Year, m.Loser.Year)
		}
		if m.Winner.RatingAfter != 1410.0 || m.Loser.RatingAfter != 1390.0 {
			t.Fatalf(
				"got winner rating %v loser rating %v, want 1410 / 1390",
				m.Winner.RatingAfter,
				m.Loser.RatingAfter,
			)
		}
		if m.CreatedAt.IsZero() {
			t.Fatal("CreatedAt is zero, want a real timestamp")
		}
	})

	t.Run("returns the matchup when the film is the loser", func(t *testing.T) {
		got, err := repo.ListFilmVotes(txCtx, fb.Id)
		if err != nil {
			t.Fatalf("ListFilmVotes: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d matchups, want 1", len(got))
		}
		// The matchup is reported from the vote's perspective (Alpha won),
		// regardless of which film was queried.
		if got[0].Winner.Id != fa.Id || got[0].Loser.Id != fb.Id {
			t.Fatalf("got winner=%d loser=%d, want winner=%d loser=%d", got[0].Winner.Id, got[0].Loser.Id, fa.Id, fb.Id)
		}
	})

	t.Run("returns nothing for a film with no votes", func(t *testing.T) {
		got, err := repo.ListFilmVotes(txCtx, 999999)
		if err != nil {
			t.Fatalf("ListFilmVotes: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %d matchups, want 0", len(got))
		}
	})

	t.Run("orders matchups most recent first", func(t *testing.T) {
		// A second, more recent matchup involving fa (this time fa loses).
		nd := seedDuel(t, txCtx, &fa, &fb)
		v2 := vote.Vote{
			DuelId:            nd.Id,
			WinnerID:          fb.Id,
			LoserId:           fa.Id,
			WinnerRatingAfter: 1405.0,
			LoserRatingAfter:  1395.0,
		}
		if err := repo.InsertVote(txCtx, &v2); err != nil {
			t.Fatalf("InsertVote: %v", err)
		}

		base := time.Now().Add(-2 * time.Hour)
		backdateVote(t, txCtx, v.Id, base)
		backdateVote(t, txCtx, v2.Id, base.Add(time.Hour))

		got, err := repo.ListFilmVotes(txCtx, fa.Id)
		if err != nil {
			t.Fatalf("ListFilmVotes: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d matchups, want 2", len(got))
		}
		// v2 is newer, so it must come first.
		if got[0].Winner.Id != fb.Id {
			t.Fatalf("got[0].Winner.Id = %d, want the newer matchup %d", got[0].Winner.Id, fb.Id)
		}
		if got[1].Winner.Id != fa.Id {
			t.Fatalf("got[1].Winner.Id = %d, want the older matchup %d", got[1].Winner.Id, fa.Id)
		}
		if got[0].CreatedAt.Before(got[1].CreatedAt) {
			t.Fatalf("matchups not ordered DESC: got[0]=%v before got[1]=%v", got[0].CreatedAt, got[1].CreatedAt)
		}
	})
}

func TestListUsersVotes(t *testing.T) {
	ctx := context.Background()
	repo := vote.NewRepository(testPool)

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txCtx := database.InjectTx(ctx, tx)

	fa, fb, d := seedFilms(t, txCtx)
	userA := seedUser(t, txCtx, "user-a-sub")
	userB := seedUser(t, txCtx, "user-b-sub")

	// userA: Alpha beats Beta.
	vA := vote.Vote{
		DuelId:            d.Id,
		UserID:            &userA,
		WinnerID:          fa.Id,
		LoserId:           fb.Id,
		WinnerRatingAfter: 1410.0,
		LoserRatingAfter:  1390.0,
	}
	if err := repo.InsertVote(txCtx, &vA); err != nil {
		t.Fatalf("InsertVote (userA): %v", err)
	}

	// userB votes the other way on a different duel.
	dB := seedDuel(t, txCtx, &fa, &fb)
	vB := vote.Vote{
		DuelId:            dB.Id,
		UserID:            &userB,
		WinnerID:          fb.Id,
		LoserId:           fa.Id,
		WinnerRatingAfter: 1405.0,
		LoserRatingAfter:  1395.0,
	}
	if err := repo.InsertVote(txCtx, &vB); err != nil {
		t.Fatalf("InsertVote (userB): %v", err)
	}

	// An anonymous vote (no user) must never be attributed to anyone.
	dAnon := seedDuel(t, txCtx, &fa, &fb)
	vAnon := vote.Vote{
		DuelId:            dAnon.Id,
		WinnerID:          fa.Id,
		LoserId:           fb.Id,
		WinnerRatingAfter: 1408.0,
		LoserRatingAfter:  1392.0,
	}
	if err := repo.InsertVote(txCtx, &vAnon); err != nil {
		t.Fatalf("InsertVote (anonymous): %v", err)
	}

	t.Run("returns only the requested user's votes with full film details", func(t *testing.T) {
		got, err := repo.ListUsersVotes(txCtx, userA)
		if err != nil {
			t.Fatalf("ListUsersVotes: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d matchups, want 1 (userB and anonymous votes must be excluded)", len(got))
		}
		m := got[0]
		if m.Winner.Id != fa.Id || m.Loser.Id != fb.Id {
			t.Fatalf("got winner=%d loser=%d, want winner=%d loser=%d", m.Winner.Id, m.Loser.Id, fa.Id, fb.Id)
		}
		if m.Winner.Title != "Alpha" || m.Loser.Title != "Beta" {
			t.Fatalf("got winner title %q loser title %q, want %q / %q", m.Winner.Title, m.Loser.Title, "Alpha", "Beta")
		}
		if m.Winner.Year != 2000 || m.Loser.Year != 2001 {
			t.Fatalf("got winner year %d loser year %d, want 2000 / 2001", m.Winner.Year, m.Loser.Year)
		}
		// Unlike ListFilmVotes, this query selects the poster paths.
		if m.Winner.ImagePath != "a.jpg" || m.Loser.ImagePath != "b.jpg" {
			t.Fatalf(
				"got winner image %q loser image %q, want %q / %q",
				m.Winner.ImagePath,
				m.Loser.ImagePath,
				"a.jpg",
				"b.jpg",
			)
		}
		if m.Winner.RatingAfter != 1410.0 || m.Loser.RatingAfter != 1390.0 {
			t.Fatalf(
				"got winner rating %v loser rating %v, want 1410 / 1390",
				m.Winner.RatingAfter,
				m.Loser.RatingAfter,
			)
		}
		if m.CreatedAt.IsZero() {
			t.Fatal("CreatedAt is zero, want a real timestamp")
		}
	})

	t.Run("scopes results per user", func(t *testing.T) {
		got, err := repo.ListUsersVotes(txCtx, userB)
		if err != nil {
			t.Fatalf("ListUsersVotes: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d matchups, want 1", len(got))
		}
		if got[0].Winner.Id != fb.Id {
			t.Fatalf("got[0].Winner.Id = %d, want %d (userB's pick)", got[0].Winner.Id, fb.Id)
		}
	})

	t.Run("returns nothing for a user with no votes", func(t *testing.T) {
		userC := seedUser(t, txCtx, "user-c-sub")
		got, err := repo.ListUsersVotes(txCtx, userC)
		if err != nil {
			t.Fatalf("ListUsersVotes: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %d matchups, want 0", len(got))
		}
	})

	t.Run("orders matchups most recent first", func(t *testing.T) {
		// A second, more recent vote by userA (this time Beta wins).
		nd := seedDuel(t, txCtx, &fa, &fb)
		v2 := vote.Vote{
			DuelId:            nd.Id,
			UserID:            &userA,
			WinnerID:          fb.Id,
			LoserId:           fa.Id,
			WinnerRatingAfter: 1402.0,
			LoserRatingAfter:  1398.0,
		}
		if err := repo.InsertVote(txCtx, &v2); err != nil {
			t.Fatalf("InsertVote: %v", err)
		}

		base := time.Now().Add(-2 * time.Hour)
		backdateVote(t, txCtx, vA.Id, base)
		backdateVote(t, txCtx, v2.Id, base.Add(time.Hour))

		got, err := repo.ListUsersVotes(txCtx, userA)
		if err != nil {
			t.Fatalf("ListUsersVotes: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d matchups, want 2", len(got))
		}
		if got[0].Winner.Id != fb.Id {
			t.Fatalf("got[0].Winner.Id = %d, want the newer matchup %d", got[0].Winner.Id, fb.Id)
		}
		if got[1].Winner.Id != fa.Id {
			t.Fatalf("got[1].Winner.Id = %d, want the older matchup %d", got[1].Winner.Id, fa.Id)
		}
		if got[0].CreatedAt.Before(got[1].CreatedAt) {
			t.Fatalf("matchups not ordered DESC: got[0]=%v before got[1]=%v", got[0].CreatedAt, got[1].CreatedAt)
		}
	})
}
