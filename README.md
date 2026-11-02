# go-auth-api

A stateless authentication API in Go. Access tokens are short-lived JWTs, refresh sessions live in Redis, and the browser receives both tokens as HttpOnly cookies. Any number of API instances can run side by side behind a load balancer.

This README documents what was built, how it works, how to run it, and the experiments and problems encountered along the way.

## Contents

1. [Stack](#stack)
2. [Architecture](#architecture)
3. [Auth design](#auth-design)
4. [API endpoints](#api-endpoints)
5. [Cookies and CSRF](#cookies-and-csrf)
6. [Configuration](#configuration)
7. [Running it](#running-it)
8. [Running multiple instances behind a gateway](#running-multiple-instances-behind-a-gateway)
9. [Inspecting Redis](#inspecting-redis)
10. [Experiments and results](#experiments-and-results)
11. [Problems we hit and how they were solved](#problems-we-hit-and-how-they-were-solved)
12. [Known limitations](#known-limitations)
13. [Production notes and next steps](#production-notes-and-next-steps)

## Stack

- Go, Gin (HTTP), GORM (Postgres)
- PostgreSQL 16 for users
- Redis 7 for refresh-token sessions
- nginx as a local gateway / load balancer (testing only)
- Docker Compose for the whole local stack

## Architecture

```
Browser / client
      |
      v
 gateway (nginx, :8000)  <-- local stand-in for a cloud load balancer / Ingress
      |
      +--> api instance 1 --+
      +--> api instance 2 --+--> Postgres  (users)
      +--> api instance N --+--> Redis     (refresh sessions)
```

The API holds no per-user data in its own memory or disk. That is what makes it stateless: any instance can serve any request, so instances can be added or removed freely (horizontal scaling).

Code layout:

```
cmd/api/main.go            startup, graceful shutdown
internal/platform/         Postgres and Redis connections
internal/modules/auth/     register, login, refresh, logout, Redis session store
internal/modules/user/     /users/me
internal/modules/health/   /health
internal/middleware/       auth (JWT), cookies, CSRF, CORS
internal/utils/            password hashing, JWT and opaque-token helpers
internal/router/           route wiring
```

## Auth design

The pattern is a short-lived JWT access token plus a rotating, opaque refresh token stored in Redis.

| Token | Format | Lifetime | Where it lives |
|---|---|---|---|
| Access token | signed JWT (HS256) with `sub` (user ID), `sid` (session ID), `jti`, `exp` | `JWT_ACCESS_TTL` (15m default) | cookie `access_token` (or `Authorization: Bearer`) |
| Refresh token | random 256-bit string, not a JWT | `JWT_REFRESH_TTL` (7 days default) | cookie `refresh_token`; only its SHA-256 hash is stored in Redis |

Normal requests only verify the JWT signature, so they need no Redis or database lookup. Redis is touched only on login, refresh and logout.

### Flows

**Login**: check the password (bcrypt), create a session, store the hash of a new refresh token in Redis, return an access token and a refresh token (as cookies).

**Authenticated request**: the middleware reads the access token (Bearer header first, then cookie), verifies signature and expiry, and puts the user ID in the request context.

**Refresh (with rotation)**:
1. Look up the hash of the presented refresh token in Redis and delete it atomically (`GETDEL`), so it can be used once.
2. Issue a new refresh token in the same session and a new access token.
3. Keep a `used:` marker for the old token.
4. If a token that was already rotated is presented again, it is treated as theft: the whole session is revoked.

**Logout**: delete the session's refresh token in Redis and clear the cookies.
**Logout everywhere**: revoke every session of the user (requires a valid access token).

### Redis data model

| Key | Value | Purpose |
|---|---|---|
| `refresh:{hash}` | `{user_id, session_id}` | an active refresh token |
| `session:{sid}` | hash of the session's current refresh token | lets a session be revoked |
| `user_sessions:{uid}` | set of session IDs | list sessions, logout everywhere |
| `used:{hash}` | the old record | reuse (theft) detection |

All keys expire with `JWT_REFRESH_TTL`.

## API endpoints

Base path: `/api/v1`.

| Method | Path | Auth | Notes |
|---|---|---|---|
| POST | `/auth/register` | no | `{name, email, password}` (password min 8) |
| POST | `/auth/login` | no | sets cookies |
| POST | `/auth/refresh` | refresh cookie (or body) | rotates the refresh token, sets cookies |
| POST | `/auth/logout` | refresh cookie (or body) | revokes the session, clears cookies |
| POST | `/auth/logout-all` | access token | revokes all of the user's sessions |
| GET | `/auth/user/:email` | no | existing endpoint, see limitations |
| GET | `/users/me` | access token | current user |
| GET | `/health` | no | pings Postgres |

Non-browser clients (mobile, scripts) can send `X-Token-Delivery: body` on login and refresh to get the tokens in the JSON response, and then authenticate with `Authorization: Bearer <access_token>`. Browsers do not get tokens in the body, so scripts on the page never see them.

## Cookies and CSRF

For browser frontends the tokens are sent as cookies because HttpOnly cookies cannot be read by JavaScript, which protects them from XSS theft.

- `access_token`: `Path=/`, `HttpOnly`, `SameSite=Lax`, lifetime equals the access TTL.
- `refresh_token`: `Path=/api/v1/auth`, `HttpOnly`, `SameSite=Strict`, lifetime equals the refresh TTL. The browser sends it only to the auth endpoints.
- Both are re-issued on every refresh, and cleared on logout or a failed refresh.

Because cookies are sent automatically, state-changing requests (POST, PUT, PATCH, DELETE) that carry our cookies must include the header `X-Requested-With: XMLHttpRequest`, otherwise the API answers `403 csrf_rejected`. A cross-site page cannot add this header without passing a CORS preflight, which only `CORS_ALLOWED_ORIGINS` passes. Requests using a Bearer header are exempt.

Frontend checklist:
- send credentials with every request (`credentials: 'include'` for fetch, `withCredentials: true` for axios);
- add `X-Requested-With: XMLHttpRequest` to POST, PUT, PATCH and DELETE;
- on a 401, call `/auth/refresh` once and retry the request; if that fails, go to the login page.

## Configuration

Copy `.env.example` to `.env`. Variables:

| Variable | Meaning |
|---|---|
| `DATABASE_URL` | Postgres URL (compose overrides it for the `api` container) |
| `REDIS_URL` | Redis URL (compose overrides it for the `api` container) |
| `REDIS_PASSWORD` | Redis password, required by compose |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Postgres container settings |
| `JWT_ACCESS_SECRET` | signing key for access tokens; must be identical on every instance |
| `JWT_ACCESS_TTL` | access-token lifetime (for example `15m`) |
| `JWT_REFRESH_TTL` | refresh-token and session lifetime (for example `168h`) |
| `COOKIE_SECURE` | `false` only for plain-HTTP local dev; **must be `true` in production** |
| `COOKIE_SAMESITE` | `lax` (default), `strict`, or `none` (only if frontend and API are on different sites) |
| `COOKIE_DOMAIN` | leave empty for host-only cookies |
| `CORS_ALLOWED_ORIGINS` | comma-separated frontend origins allowed to send credentials |
| `SHUTDOWN_TIMEOUT` | how long in-flight requests get to finish on shutdown (default `10s`) |
| `PORT` | listen port (default `8080`) |

`JWT_REFRESH_SECRET` is no longer used, because refresh tokens are random strings rather than JWTs.

## Running it

Everything in Docker (API, Postgres, Redis, gateway):

```
docker compose up --build -d
```

The API is then reachable through the gateway at `http://localhost:8000/api/v1`. The API containers do not publish their own port.

Run the app on the host with air while Postgres and Redis run in Docker:

```
docker compose up -d postgres redis
air
```

Host ports used by compose: Postgres `5432`, Redis `127.0.0.1:6380` (not 6379, see problems below), gateway `8000`.

`docker compose down` stops and removes the containers but keeps the data (named volumes `pg-data`, `redis-data`). `docker compose down -v` also deletes the data.

Rebuild after any Go code change: `docker compose up --build -d`.

### Quick manual test with curl

```
B=localhost:8000/api/v1; H='content-type: application/json'; X='X-Requested-With: XMLHttpRequest'; J=jar
curl -XPOST $B/auth/register -H "$H" -d '{"name":"t","email":"t@x.com","password":"password1"}'
curl -c $J -XPOST $B/auth/login -H "$H" -d '{"email":"t@x.com","password":"password1"}'
curl -b $J $B/users/me
curl -b $J -c $J -XPOST $B/auth/refresh -H "$X"
curl -b $J -c $J -XPOST $B/auth/logout -H "$X"
```

## Running multiple instances behind a gateway

```
docker compose up -d --scale api=3
```

All traffic goes through the nginx gateway at port 8000. Each response includes an `X-Served-By` header with the IP of the instance that answered; `docker compose logs -f api` shows the requests spread over `api-1`, `api-2`, `api-3`.

Useful commands:

```
docker compose ps                     # which containers are running
docker compose logs -f api            # logs of all API instances
docker compose up -d --scale api=2 api   # scale without starting the gateway
docker compose stop gateway           # stop only the gateway
```

[nginx.conf](nginx.conf) re-resolves the `api` name through Docker DNS every 5 seconds, so scaling up or down needs no gateway restart.

## Inspecting Redis

```
docker compose exec redis sh -c 'redis-cli -a "$REDIS_PASSWORD" --no-auth-warning'
```

Useful commands: `KEYS *`, `KEYS refresh:*`, `GET refresh:<hash>`, `TTL <key>`, `SMEMBERS user_sessions:<user-id>`, `MONITOR` (live view of commands). Keys contain hashes, not real tokens. A GUI client can connect to `127.0.0.1:6380` with the password from `.env`.

## Experiments and results

All experiments were run against the Docker stack with `curl` and a cookie jar.

**1. Token flow with Redis sessions**

| Step | Result |
|---|---|
| register, login, `GET /users/me` | works |
| refresh | returns a new refresh token (rotation) |
| reuse the old refresh token | 401, and the session is revoked |
| use the newer token after that reuse | also rejected |
| logout, then use the refresh token | rejected |
| `logout-all` without a token / with a token | 401 / works |

**2. Cookie mode**

| Step | Result |
|---|---|
| login | both cookies set, response `data: null` |
| `/users/me` with only the cookie | works |
| refresh without `X-Requested-With` | 403 `csrf_rejected` |
| refresh with the header | cookies re-issued |
| login with `X-Token-Delivery: body` | tokens included in the JSON |
| logout, then `/users/me` and refresh | both rejected |

**3. Statelessness across instances (two ad-hoc containers on ports 8080 and 8081)**

Login on one instance, `/users/me` on the other, refresh on each in turn, logout on one and replay the refresh token on the other. Everything behaved as one system. The replay after logout was rejected on the other instance, which shows revocation is shared through Redis.

**4. Three instances behind the nginx gateway**

Login, `/users/me`, refresh and logout all succeeded while requests were served by different instances (visible in `X-Served-By`). After logout, `/users/me` returned 401 from a different instance.

**5. Graceful shutdown**

While a burst of logins ran through the gateway, one instance was stopped with `docker stop`. It logged `shutdown signal received, draining requests`, finished its in-flight login (200), logged `server stopped`, and exited with code 0 in about 0.25 seconds.

**6. Scaling without restarting the gateway**

After switching nginx to per-request DNS resolution, scaling from 1 to 2 to 3 instances was picked up within about 7 seconds, and stopping one instance caused no stalls (18 requests, all 200).

## Problems we hit and how they were solved

| Problem | Cause | Fix |
|---|---|---|
| Redis container would not start: port already in use | a Redis installed on the host was already listening on 6379 | the container is published on host port **6380** (`127.0.0.1:6380`), and `REDIS_URL` for local runs uses 6380. The host Redis was left untouched. |
| Tokens stolen from a refresh JWT could not be revoked | the original refresh token was a stateless JWT | refresh tokens are now opaque random strings stored hashed in Redis, rotated on every use, with reuse detection |
| Test of "refresh after logout" looked passing but proved nothing | the cookie jar had already been cleared by logout, so no token was sent | re-tested by replaying the raw refresh token against another instance |
| Cannot scale `api`: port 8080 conflict | every replica tried to publish host port 8080 | the `api` service now uses `expose` instead of `ports`; only the gateway publishes a port |
| Requests hung (about 60 s) after an instance was removed | nginx resolved `api` once at startup and kept sending traffic to the dead IP | `resolver 127.0.0.11 valid=5s`, a variable in `proxy_pass` to force re-resolution, a 2 s connect timeout and `proxy_next_upstream error timeout` to retry another instance |
| Stopping the wrong container during a test | a container name filter matched `redis` because the name contains "api" | Redis was restarted immediately; no data was lost because Redis uses AOF persistence (`appendonly yes`) on the `redis-data` volume |
| `docker stop` cut requests off | `router.Run()` exits immediately on SIGTERM | an `http.Server` with signal handling and `Shutdown` with a timeout; `stop_grace_period: 15s` in compose gives it room |

## Known limitations

- **Access tokens survive logout.** After logout the access token stays valid until it expires (15 minutes by default), because normal requests never check Redis. Keep the TTL short, or add a per-session check at the cost of one Redis read per request.
- **Concurrent refresh race.** Two requests refreshing with the same token at nearly the same moment: the second is rejected, and if it arrives after the first has finished it is treated as reuse and revokes the session. A short grace window would fix this; it is not implemented.
- **Refresh does not re-check the user.** A deleted user keeps working until the session ends.
- **Single Redis container.** No replication. If Redis is down, login and refresh fail, but access tokens already issued keep working until they expire.
- **The gateway gap with DNS discovery.** nginx caches DNS for up to 5 seconds, so a request can still reach an instance that has just stopped. The retry settings keep the cost to roughly 2 seconds, but cannot make it zero. Closing the gap fully needs the instance to fail its readiness check and keep serving for a few seconds before stopping, which Kubernetes and cloud load balancers support. We discussed it but did not implement it.
- **Health check is minimal.** `/health` only pings Postgres. There is no separate liveness check and Redis is not checked.
- **`GET /auth/user/:email` is unauthenticated** and returns the ID and name for any email. Consider protecting or removing it.
- **`AutoMigrate` runs on every start.** With many instances starting together this can race; run migrations as a separate step in production.
- **Schema changes are not versioned.** There are no migration files yet.

## Production notes and next steps

- Set `COOKIE_SECURE=true`, a strong `JWT_ACCESS_SECRET` and a strong `REDIS_PASSWORD`.
- Replace the local nginx with the platform's load balancer or Ingress (it handles HTTPS and balancing). `nginx.conf` and the `gateway` service are for local testing only.
- Use managed or replicated Postgres and Redis.
- Make the load balancer pass the real client IP and configure Gin's trusted proxies accordingly.
- Kubernetes autoscaling would need: a Deployment with CPU and memory requests, a Service, a HorizontalPodAutoscaler, an Ingress, Secrets and ConfigMaps, a migration Job, and liveness and readiness probes.
- If other services need to verify tokens, switch to an asymmetric algorithm (RS256 or EdDSA) so only the auth service holds the private key.
- Add rate limiting on login and refresh, and tests.

Horizontal scaling here means running more copies of the same stateless app behind a load balancer. The shared parts (Postgres and Redis) are the next bottleneck and are scaled separately.
