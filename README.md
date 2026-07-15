# FilmMash

The app shows you two movies side by side, you pick the better one, and an Elo rating system continuously re-ranks the entire catalog based on the outcome of every duel. Films are seeded from the [TMDB](https://www.themoviedb.org/) top-rated list.

## How it works

1. A **duel** is created by pairing two random films (or the previous winner against a new challenger).
2. **Winner-stays matchmaking**: the challenger is drawn from the films closest to the winner in rating — at most `MAXIMUM_CANDIDATES`, filtered to a ±`DUEL_RATING_WINDOW` rating window, then padded back up to `MINIMUM_CANDIDATES` with the next-closest films when the window is too sparse (so runaway leaders don't just duel each other forever). The draw is weighted by TMDB popularity (`DUEL_POPULARITY_WEIGHT`), and films with fewer than `DUEL_NEW_FILM_THRESHOLD` duels get a `DUEL_NEW_FILM_BOOST` multiplier so newcomers get rated quickly. All knobs are documented in [.env.example](.env.example).
3. The user votes for one of them. The vote is registered inside a single database transaction that locks both films' ratings (`SELECT ... FOR UPDATE`), computes the new Elo scores, stores the vote, and updates both films atomically.
4. Ratings use the classic Elo formula with `K = 20` and a starting rating of `1400`.
5. Anyone can vote anonymously; logged-in users get a personal vote history, and admins get a user-management dashboard.

## Tech stack

| Concern | Technology |
|---|---|
| Language | Go 1.25+ |
| HTTP router | [chi v5](https://github.com/go-chi/chi) |
| Frontend | Server-rendered Go templates + [HTMX](https://htmx.org/) (templates embedded in the binary via `go:embed`) |
| Database | PostgreSQL 18, [pgx v5](https://github.com/jackc/pgx) connection pool |
| Query generation | [sqlc](https://sqlc.dev/) — type-safe Go generated from SQL in `internal/database/queries/` |
| Migrations | [goose](https://github.com/pressly/goose) |
| Authentication | OAuth 2.0 Authorization Code flow + PKCE with OpenID Connect ([go-oidc](https://github.com/coreos/go-oidc) + `golang.org/x/oauth2`); any OIDC-compliant IdP works via discovery — [Zitadel](https://zitadel.com/) in this deployment |
| Logging | `log/slog` structured JSON logs |
| Metrics | Prometheus ([client_golang](https://github.com/prometheus/client_golang)) with custom HTTP, duel, vote, film, and system metrics |
| Observability | Grafana, Prometheus, Alertmanager, Loki, Grafana Alloy (local profile) or Grafana Cloud (production) |
| Reverse proxy | nginx |
| Packaging | Docker multi-stage builds + Docker Compose |
| External API | TMDB (film/director seed data) |

## Architecture

The codebase is organized by **vertical slices**: instead of global `handlers/`, `services/`, and `repositories/` layers, each feature (`film`, `duel`, `vote`, `auth`, `admin`) lives in its own package containing everything that feature needs — handler, service, repository, domain types, and errors. Layering (**handler → service → repository**) exists *inside* each slice, cross-slice calls go through service interfaces, and everything is wired together with constructor injection in [cmd/filmmash-go/server.go](cmd/filmmash-go/server.go).

```
cmd/
├── filmmash-go/        # main server: config, DB pool, DI wiring, HTTP server
└── seed/               # CLI that pulls TMDB top-rated films into Postgres

internal/
├── web/                # chi router: routes, route groups, middleware chain
│
│  # vertical slices — each owns its handler, service, repository, and errors
├── film/               # film catalog, search, pagination, Elo calculation
├── duel/               # duel creation (random pairing / winner-stays matchmaking)
├── vote/               # vote registration (transactional Elo update), vote listings
├── auth/               # OAuth2/OIDC flow (Auth Code + PKCE), DB-backed sessions, middleware
├── admin/              # admin dashboard (user management through the IdP's API)
│
│  # shared infrastructure
├── zitadel/            # thin IdP adapter: role-claim mapper + M2M management client
├── database/           # pgx pool, transaction manager, sqlc-generated code (dbgen/)
│   └── queries/        # SQL sources consumed by sqlc
├── view/               # go:embed'ed HTML templates + parsed template cache
├── middleware/         # request ID, response logger, panic recoverer
├── metrics/            # Prometheus registry + domain metric collectors
├── config/             # env-based configuration, AES key handling
├── tmdb/               # TMDB HTTP client
└── testdb/             # test database bootstrap for integration tests

migrations/             # goose SQL migrations (schema source of truth for sqlc)
monitoring/             # Prometheus, Alertmanager, Loki, Alloy, Grafana configs
nginx/                  # reverse proxy config
```

### Request flow

```
nginx (:8080) ──► chi router ──► middleware (client IP, request ID,
                                  logging, metrics, recover)
                       │
                       ▼
                handler (renders HTMX/HTML fragments)
                       │
                       ▼
                service (business rules, transactions via TxManager)
                       │
                       ▼
                repository (sqlc-generated queries over pgx)
                       │
                       ▼
                   PostgreSQL
```

- **Transactions** are managed by a `TxManager` that carries the transaction in `context.Context`, so services can compose repository calls from different domains (e.g. a vote updates the `votes` and `films` tables in one transaction).
- **Views** are classic Go templates rendered server-side; HTMX swaps fragments (duel cards, film lists, search results, modals) without a SPA framework. All templates are parsed once at startup into a template cache and embedded in the binary.
- **Errors** carry domain sentinel errors per package, mapped to HTTP status codes at the handler layer.

### Authentication & sessions

The bulk of the auth code is a provider-agnostic **OAuth 2.0 / OpenID Connect** implementation in [internal/auth](internal/auth/):

- **Authorization Code flow with PKCE** (S256 challenge), plus `state` and `nonce` generation and verification on the callback. ID tokens are verified against the issuer's signing keys; user info comes from the standard `userinfo` endpoint.
- Endpoints are resolved through **OIDC discovery**, including the `end_session_endpoint` used for RP-initiated logout (`id_token_hint`) — so any spec-compliant identity provider can be plugged in via configuration.
- The in-flight login state (state/nonce/PKCE verifier) is stored in an AES-GCM-encrypted cookie (`AES_KEY`), keeping the flow stateless server-side.
- **Sessions** are first-party and stored in Postgres: the cookie holds an opaque token whose SHA-256 hash is persisted; access/refresh/ID tokens are encrypted at rest. Session events are audit-logged, and a background job cleans up expired sessions every 5 minutes.
- **Role-based access**: `/admin` routes require the `admin` role. Roles are extracted from token claims by a pluggable `RoleMapper` injected into the provider.

The IdP-specific code is deliberately thin and isolated in [internal/zitadel](internal/zitadel/): a `RoleMapper` that reads Zitadel's project-roles claim, and a machine-to-machine (client credentials) token source used by the admin slice to call the IdP's management API. Swapping identity providers means replacing this adapter, not the auth slice.

### Database schema

Core tables: `directors`, `films` (with current Elo `rating`), `duels`, `votes` (with post-vote ratings and a monotonic `applied_seq`), `users`, `sessions`, and `session_events`. The schema lives in [migrations/](migrations/) and doubles as sqlc's schema input, so generated code is always in sync with migrations.

### Observability

- The app exposes `/metrics` (Prometheus) and `/health` (blocked from the outside by nginx).
- **Local/dev** (`COMPOSE_PROFILES=monitoring`): full self-hosted stack — Prometheus (:9090), Alertmanager (:9093, email alerts), Loki (:9080), Grafana (:9000) with provisioned dashboards, and Alloy scraping metrics and shipping container logs.
- **Production** ([docker-compose.prod.yaml](docker-compose.prod.yaml)): only Alloy runs, forwarding metrics and logs to Grafana Cloud.

## Running it

### Prerequisites

- Docker + Docker Compose
- A TMDB API token and an OIDC identity provider (Zitadel in the default config — see [.env.example](.env.example))

### Quick start

```sh
cp .env.example .env   # fill in TMDB + identity provider credentials
docker compose up -d   # or: make compose-up
```

Compose brings up, in order:

1. `postgres` — PostgreSQL 18 (host port 5440)
2. `migrate` — runs goose migrations, then exits
3. `seed` — downloads ~500 top-rated films from TMDB, then exits
4. `app` — the Go server
5. `nginx` — reverse proxy on **http://localhost:8080**
6. The monitoring stack, if the `monitoring` profile is enabled

For production, `make compose-prod` layers [docker-compose.prod.yaml](docker-compose.prod.yaml) on top to ship telemetry to Grafana Cloud instead of running the local monitoring stack.

## Development

```sh
make test            # go test ./... (integration tests need a test DB, see below)
make sqlc            # regenerate internal/database/dbgen from SQL queries
make new-migration   # create a goose migration (MIGRATION_NAME=<name>)
make migrate         # apply migrations (uses GOOSE_* vars from .env)
make migrate-test-db # apply migrations to the test database (.env.test)
```

Repository tests are integration tests that run against a real Postgres instance configured in `.env.test`. As a safety net, the test helper refuses to run unless the database URL contains the word `test`.

## Main routes

| Route | Description |
|---|---|
| `GET /ui` | Duel page (vote on a pair of films) |
| `POST /ui/duel/{duel_id}/result` | Register a vote (requires session) |
| `GET /ui/films` | Ranked film list (keyset-paginated, HTMX infinite scroll) |
| `GET /ui/films/search` | Film search |
| `GET /ui/film/{id}` | Film detail |
| `GET /ui/my_votes` | Logged-in user's vote history |
| `GET /auth/login` · `/auth/callback` · `/auth/logout` | OIDC flow |
| `GET /admin/users` | Admin user management (role-gated) |
| `GET /health` · `GET /metrics` | Health check / Prometheus metrics (internal) |
