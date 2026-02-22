# NavPlane Project Tracker

## Overview

NavPlane is a passthrough proxy for AI providers that tracks usage and enables policy-based routing. Users replace their AI provider URL with NavPlane's URL and use NavPlane API keys.

## Task Status

### Completed

#### Task 1: PostgreSQL Setup ✅
**Branch:** `ronald/01-postgres-setup` (merged)

- [x] Add PostgreSQL service to docker-compose
- [x] Create initial schema migration (organizations, provider_keys, request_logs)
- [x] Add golang-migrate for schema management
- [x] Auto-run migrations on server startup
- [x] Add database connection pooling configuration
- [x] Set up CI pipeline (lint, test, docker build)
- [x] Add `updated_at` triggers for PostgreSQL
- [x] Remove exposed PostgreSQL port for security

#### Task 2: Org Model & Basic Auth ✅
**Branch:** `ronald/02-org-model` (merged)

- [x] Create org package with manager/datastore pattern
- [x] Implement `Org` model (ID, Name, APIKeyHash, Enabled, timestamps)
- [x] Implement API key generation (`np_<uuid>` format)
- [x] Implement SHA-256 key hashing for storage
- [x] Create auth middleware (Bearer token extraction)
- [x] Wire auth middleware to `/v1/chat/completions`
- [x] Add comprehensive tests with sqlmock
- [x] Update AGENTS.md with manager/datastore guidelines

#### Task 3: Org-Level Kill Switch & Admin API ✅
**Branch:** `ronald/02-org-model` (in review)

- [x] Add `enabled` flag check in auth middleware
- [x] Admin endpoint to toggle org status (`PUT /admin/orgs/{id}/enabled`)
- [x] Disabled org requests fail immediately with 403
- [x] No provider calls made for disabled orgs
- [x] Re-enabling restores traffic instantly
- [x] Full CRUD endpoints for orgs (`GET/POST /admin/orgs`, `GET/PUT/DELETE /admin/orgs/{id}`)
- [x] API key rotation endpoint (`POST /admin/orgs/{id}/rotate-key`)
- [x] Admin handler tests with sqlmock

#### Task 4: BYOK + Users + Provider System ✅
**Branch:** `ronald/04-byok-users-providers` (in review)

Implemented provider key storage, user management (via Auth0), and provider interface.

**Database (Migration 000002):**
- [x] `user_identities` table (auth0_user_id, email, is_admin)
- [x] `org_members` table (user-org many-to-many with roles)
- [x] `org_provider_settings` table (enable/disable providers per org)
- [x] Update `provider_keys` schema for envelope encryption (encrypted_dek, dek_nonce)

**Auth0 Integration (`jwtauth` package):**
- [x] JWT verification with JWKS caching
- [x] Extract user claims from JWT (sub, email, name, permissions)
- [x] `Claims` type with helper methods
- [x] JWT middleware for handlers

**User Management (`user` package):**
- [x] User identity model and datastore
- [x] Upsert user identity from JWT claims
- [x] Org membership management (add/remove members)
- [x] Role-based access (owner, member)
- [x] Admin access check (admins can access any org)

**Provider System (`provider` package):**
- [x] Provider interface (Name, BaseURL, Models, ValidateKey, AuthHeader)
- [x] OpenAI provider implementation (gpt-4o, gpt-4o-mini, o1, etc.)
- [x] Anthropic provider implementation (claude-3.5-sonnet, opus, haiku)
- [x] Provider registry with auto-registration

**BYOK Encryption (`providerkey` package):**
- [x] Envelope encryption (AES-256-GCM for both KEK→DEK and DEK→key)
- [x] Key rotation support via `ReEncryptDEK()` method
- [x] Validate provider key before storage (calls provider API)
- [x] Decrypt only in memory at request time
- [x] Never expose encrypted data in JSON responses

**Org Provider Settings (`orgsettings` package):**
- [x] Enable/disable providers per org
- [x] Allow/block specific models per org
- [x] Model access check with priority rules

**Config Updates:**
- [x] Removed `PROVIDER_BASE_URL` and `PROVIDER_API_KEY`
- [x] Added `ENCRYPTION_KEY`, `AUTH0_DOMAIN`, `AUTH0_AUDIENCE`
- [x] Optional `ENCRYPTION_KEY_NEW` for rotation

---

### In Progress

_(None currently)_

---

### Pending

#### Task 5: Header Swapping + Proxy Director (Mostly Complete)
**Priority:** High

Wire up provider keys to the proxy and rewrite requests.

- [x] Extract org from request context (via auth middleware)
- [x] Load decrypted provider key from vault
- [x] Inject provider-specific `Authorization` header (using Provider interface)
- [x] Rewrite request URL to provider endpoint (using Provider.BaseURL)
- [x] Fail immediately if key is missing with clear error
- [x] Model-based provider routing (gpt-* → OpenAI, claude-* → Anthropic)
- [ ] Add orgsettings check (provider/model restrictions)
- [ ] Check org.Enabled in proxy (currently only in API key auth)

#### Task 7: Async Request Logging
**Priority:** Medium

Log basic request metadata asynchronously without impacting latency.

- [ ] Define request log schema (timestamp, org, provider, model, status, latency, tokens)
- [ ] Async write to Postgres (buffered channel)
- [ ] Logging never blocks proxy
- [ ] Logs persist even under load
- [ ] Proxy continues if logging fails

#### Task 8: Minimal Admin API (Mostly Complete)
**Priority:** Medium

Expose admin endpoints used by the dashboard.

- [x] Enable/disable org (`PUT /admin/orgs/{id}/enabled`)
- [x] Org CRUD endpoints
- [x] API key rotation
- [x] Create/delete provider keys (`GET/POST /admin/orgs/{id}/provider-keys`)
- [x] List providers (`GET /providers`)
- [ ] List recent requests (depends on Task 7)
- [ ] Auth-protected endpoints (JWT middleware for admin routes)

#### Task 9: Streaming (SSE) Passthrough Support ✅
**Status:** Complete (implemented in Task 2)

Support streaming chat completions using Server-Sent Events.

- [x] Detect `stream: true` in request
- [x] Use `http.Flusher` to stream responses
- [x] Handle client disconnects gracefully
- [x] No full-response buffering
- [x] Client disconnect terminates upstream request

#### Task 10: Documentation MVP
**Priority:** Low

Document how to use NavPlane MVP.

- [ ] Environment setup guide
- [ ] How to integrate (baseURL swap)
- [ ] Known limitations
- [ ] New dev can integrate in <10 minutes

#### Task 11: E2E Validation
**Priority:** Low

Validate the MVP end-to-end with a real OpenAI client.

- [ ] Demo app using OpenAI SDK
- [ ] Switch baseURL to NavPlane
- [ ] Test streaming
- [ ] Test key rotation
- [ ] Test kill switch
- [ ] Test logging

---

## Development Workflow

1. Work on one major task at a time
2. Create branch: `ronald/<task-number>-<name>`
3. Write tests for all new code
4. Use semantic commits (`feat:`, `fix:`, `test:`, `docs:`, `refactor:`, `chore:`)
5. Push to remote for review before merging
6. Update this file when tasks complete

## Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| Manager/Datastore pattern | Clean separation of business logic and persistence |
| SHA-256 hashed API keys | Never store plaintext NavPlane keys |
| `np_` key prefix | Easy identification and validation |
| PostgreSQL triggers for `updated_at` | Auto-update timestamps on modifications |
| No exposed DB port | Security best practice |
| sqlmock for testing | Unit test DB operations without real database |
| Auth0 for user management | Delegate passwords, MFA, social login - not core to product |
| JWT verification only | No session storage, stateless auth for dashboard |
| BYOK (Bring Your Own Key) | Orgs provide their own provider API keys |
| Envelope encryption | Support master key rotation without re-encrypting all data |
| Hardcoded provider URLs | No config needed - OpenAI/Anthropic URLs are stable |
| Provider interface | Extensible design for adding new providers |
| Validate keys on creation | Fail fast - don't store invalid provider keys |
