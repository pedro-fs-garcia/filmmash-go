package auth_test

import (
	"context"
	"errors"
	"filmmash/internal/auth"
	"filmmash/internal/database"
	"filmmash/internal/testdb"
	"log"
	"net/netip"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, cleanup, err := testdb.New(ctx)
	if err != nil {
		log.Fatalf("test db setup: %v", err)
	}
	testPool = pool
	code := m.Run()
	cleanup()
	os.Exit(code)
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

func newSessionDB(userID uuid.UUID, tokenHash string) auth.SessionDB {
	scopes := "openid profile email"
	userAgent := "test-agent/1.0"
	accessExpiry := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	return auth.SessionDB{
		TokenHash:            []byte(tokenHash),
		UserID:               userID,
		AccessToken:          []byte("access-token-bytes"),
		AccessTokenExpiresAt: &accessExpiry,
		RefreshToken:         []byte("refresh-token-bytes"),
		IDToken:              []byte("id-token-bytes"),
		Scopes:               &scopes,
		IPAddress:            netip.MustParseAddr("198.51.100.7"),
		UserAgent:            &userAgent,
		ExpiresAt:            time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond),
	}
}

func assertSessionMatches(t *testing.T, got auth.Session, want auth.SessionDB) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("session ID = %v, want %v", got.ID, want.ID)
	}
	if got.UserID != want.UserID {
		t.Errorf("session UserID = %v, want %v", got.UserID, want.UserID)
	}
	if got.AccessToken != string(want.AccessToken) {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, want.AccessToken)
	}
	if got.RefreshToken != string(want.RefreshToken) {
		t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, want.RefreshToken)
	}
	if got.IDToken != string(want.IDToken) {
		t.Errorf("IDToken = %q, want %q", got.IDToken, want.IDToken)
	}
	wantScopes := strings.Split(*want.Scopes, " ")
	if !reflect.DeepEqual(got.Scopes, wantScopes) {
		t.Errorf("Scopes = %v, want %v", got.Scopes, wantScopes)
	}
	if got.UserAgent != *want.UserAgent {
		t.Errorf("UserAgent = %q, want %q", got.UserAgent, *want.UserAgent)
	}
	if got.IPAddress.String() != want.IPAddress.String() {
		t.Errorf("IPAddress = %v, want %v", got.IPAddress, want.IPAddress)
	}
	if !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, want.ExpiresAt)
	}
	if want.AccessTokenExpiresAt != nil && !got.AccessTokenExpiresAt.Equal(*want.AccessTokenExpiresAt) {
		t.Errorf("AccessTokenExpiresAt = %v, want %v", got.AccessTokenExpiresAt, *want.AccessTokenExpiresAt)
	}
}

func TestRepository(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := auth.NewRepository(testPool)

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	txCtx := database.InjectTx(ctx, tx)

	// alice is populated by the UpsertUser subtest and reused as the owner of
	// the sessions exercised further down. Subtests run in order, so later
	// subtests can rely on it being set.
	var alice auth.User

	t.Run("UpsertUser", func(t *testing.T) {
		alice = auth.User{ZitadelSub: "auth-test|alice"}
		if err := repo.UpsertUser(txCtx, &alice); err != nil {
			t.Fatalf("UpsertUser: %v", err)
		}
		if !IsValidUUID(t, alice.ID.String()) {
			t.Fatalf("UpsertUser id = %v, want a v7 UUID", alice.ID)
		}
		if alice.CreatedAt.IsZero() {
			t.Error("UpsertUser did not populate CreatedAt")
		}
	})

	t.Run("UpsertUser is idempotent on conflict", func(t *testing.T) {
		again := auth.User{ZitadelSub: alice.ZitadelSub}
		if err := repo.UpsertUser(txCtx, &again); err != nil {
			t.Fatalf("UpsertUser on existing sub: %v", err)
		}
		if again.ID != alice.ID {
			t.Errorf("UpsertUser conflict id = %v, want %v", again.ID, alice.ID)
		}
		if !again.CreatedAt.Equal(alice.CreatedAt) {
			t.Errorf("UpsertUser conflict created_at = %v, want %v", again.CreatedAt, alice.CreatedAt)
		}
	})

	t.Run("GetUserBySub", func(t *testing.T) {
		got, err := repo.GetUserBySub(txCtx, alice.ZitadelSub)
		if err != nil {
			t.Fatalf("GetUserBySub: %v", err)
		}
		if got.ID != alice.ID || got.ZitadelSub != alice.ZitadelSub {
			t.Errorf("GetUserBySub = %+v, want id %v sub %q", got, alice.ID, alice.ZitadelSub)
		}
	})

	t.Run("GetUserBySub unknown sub should fail", func(t *testing.T) {
		_, err := repo.GetUserBySub(txCtx, "auth-test|nobody")
		if err == nil {
			t.Fatal("got no error, want ErrNotFound")
		}
		if !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("GetUserById", func(t *testing.T) {
		got, err := repo.GetUserById(txCtx, alice.ID)
		if err != nil {
			t.Fatalf("GetUserById: %v", err)
		}
		if got.ID != alice.ID || got.ZitadelSub != alice.ZitadelSub {
			t.Errorf("GetUserById = %+v, want id %v sub %q", got, alice.ID, alice.ZitadelSub)
		}
	})

	t.Run("GetUserById unknown id should fail", func(t *testing.T) {
		_, err := repo.GetUserById(txCtx, uuid.New())
		if err == nil {
			t.Fatal("got no error, want ErrNotFound")
		}
		if !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("Insert", func(t *testing.T) {
		s := newSessionDB(alice.ID, "token-hash-insert")
		if err := repo.Insert(txCtx, &s); err != nil {
			t.Fatalf("Insert: %v", err)
		}
		if !IsValidUUID(t, s.ID.String()) {
			t.Fatalf("Insert id = %v, want a v7 UUID", s.ID)
		}

		got, err := repo.GetByTokenHash(txCtx, s.TokenHash)
		if err != nil {
			t.Fatalf("GetByTokenHash: %v", err)
		}
		assertSessionMatches(t, got, s)
	})

	t.Run("Insert duplicate token_hash should fail", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		first := newSessionDB(alice.ID, "token-hash-dup")
		if err := repo.Insert(nctx, &first); err != nil {
			t.Fatalf("first Insert: %v", err)
		}

		dup := newSessionDB(alice.ID, "token-hash-dup")
		err = repo.Insert(nctx, &dup)
		if err == nil {
			t.Fatal("got no error, want ErrDuplicateEntry")
		}
		if !errors.Is(err, database.ErrDuplicateEntry) {
			t.Fatalf("got %v, want ErrDuplicateEntry", err)
		}
	})

	t.Run("Insert with unknown user should fail", func(t *testing.T) {
		ntx, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("begin savepoint: %v", err)
		}
		defer ntx.Rollback(ctx)
		nctx := database.InjectTx(ctx, ntx)

		s := newSessionDB(uuid.New(), "token-hash-orphan")
		err = repo.Insert(nctx, &s)
		if err == nil {
			t.Fatal("got no error, want a foreign key violation")
		}
	})

	t.Run("GetByTokenHash unknown hash should fail", func(t *testing.T) {
		_, err := repo.GetByTokenHash(txCtx, []byte("token-hash-missing"))
		if err == nil {
			t.Fatal("got no error, want ErrNotFound")
		}
		if !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("DeleteByTokenHash", func(t *testing.T) {
		s := newSessionDB(alice.ID, "token-hash-delete")
		if err := repo.Insert(txCtx, &s); err != nil {
			t.Fatalf("Insert: %v", err)
		}

		if err := repo.DeleteByTokenHash(txCtx, s.TokenHash); err != nil {
			t.Fatalf("DeleteByTokenHash: %v", err)
		}

		_, err := repo.GetByTokenHash(txCtx, s.TokenHash)
		if !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("GetByTokenHash after delete = %v, want ErrNotFound", err)
		}
	})

	t.Run("DeleteByTokenHash unknown hash should fail", func(t *testing.T) {
		err := repo.DeleteByTokenHash(txCtx, []byte("token-hash-never-existed"))
		if err == nil {
			t.Fatal("got no error, want ErrSessionNotFound")
		}
		if !errors.Is(err, auth.ErrSessionNotFound) {
			t.Fatalf("got %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("InsertEvent", func(t *testing.T) {
		e := auth.SessionEvent{
			SessionID:  uuid.New(),
			ZitadelSub: alice.ZitadelSub,
			Event:      auth.EventCreated,
			IPAddress:  netip.MustParseAddr("203.0.113.5"),
		}
		if err := repo.InsertEvent(txCtx, &e); err != nil {
			t.Fatalf("InsertEvent: %v", err)
		}
		if e.ID == 0 {
			t.Error("InsertEvent did not populate ID")
		}
		if e.CreatedAt.IsZero() {
			t.Error("InsertEvent did not populate CreatedAt")
		}
	})
}
