# Lectr Project Tracker

## Overview

Lectr is a passthrough proxy for AI providers that tracks usage and enables policy-based routing. Users replace their AI provider URL with Lectr's URL and use Lectr API keys.

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
- [x] Implement API key generation (`lc_<uuid>` format)
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

---

### In Progress

_(None currently)_

---

### Pending

#### Task 4: BYOK + Users + Provider System
**Priority:** High

Implement provider key storage, user management (via Auth0), and provider interface.

**Database (Migration 000002):**
- [ ] `user_identities` table (auth0_user_id, email, is_admin)
- [ ] `org_members` table (user-org many-to-many with roles)
- [ ] `org_provider_settings` table (enable/disable providers per org)
- [ ] Update `provider_keys` schema for envelope encryption

**Auth0 Integration:**
- [ ] JWT verification middleware (JWKS)
- [ ] Extract user claims from JWT
- [ ] Upsert user identity on login
- [ ] Check org membership for authorization

**Provider System:**
- [ ] Provider interface (Name, BaseURL, Models, ValidateKey)
- [ ] OpenAI provider implementation
- [ ] Anthropic provider implementation
- [ ] Provider registry for lookup

**BYOK Encryption:**
- [ ] Envelope encryption (KEK encrypts DEK, DEK encrypts key)
- [ ] Key rotation support via `ENCRYPTION_KEY_NEW`
- [ ] Validate provider key before storage
- [ ] Decrypt only in memory at request time

**Org Provider Settings:**
- [ ] Enable/disable providers per org
- [ ] Allow/block specific models per org

#### Task 5: Header Swapping + Proxy Director
**Priority:** High

Wire up provider keys to the proxy and rewrite requests.

- [ ] Extract org from request context
- [ ] Load decrypted provider key from vault
- [ ] Inject provider-specific `Authorization` header
- [ ] Remove Lectr auth headers before forwarding
- [ ] Rewrite request URL to provider endpoint
- [ ] Fail immediately if key is missing or provider disabled

#### Task 6: Reverse Proxy Director
**Priority:** Medium

Create a Director function that rewrites outbound requests before forwarding.

- [ ] Rewrite request URL to provider endpoint
- [ ] Strip Lectr auth headers
- [ ] Inject provider headers
- [ ] Preserve request body and method
- [ ] Provider responses pass back unchanged

#### Task 7: Async Request Logging
**Priority:** Medium

Log basic request metadata asynchronously without impacting latency.

- [ ] Define request log schema (timestamp, org, provider, model, status, latency, tokens)
- [ ] Async write to Postgres (buffered channel)
- [ ] Logging never blocks proxy
- [ ] Logs persist even under load
- [ ] Proxy continues if logging fails

#### Task 8: Minimal Admin API (Partially Complete)
**Priority:** Medium

Expose admin endpoints used by the dashboard.

- [x] Enable/disable org (`PUT /admin/orgs/{id}/enabled`)
- [x] Org CRUD endpoints
- [x] API key rotation
- [ ] Create/delete provider keys (depends on Task 4)
- [ ] List recent requests (depends on Task 7)
- [ ] Auth-protected endpoints

#### Task 9: Streaming (SSE) Passthrough Support
**Priority:** Medium

Support streaming chat completions using Server-Sent Events.

- [ ] Detect `stream: true` in request
- [ ] Use `http.Flusher` to stream responses
- [ ] Handle client disconnects gracefully
- [ ] No full-response buffering
- [ ] Client disconnect terminates upstream request

#### Task 10: Documentation MVP
**Priority:** Low

Document how to use Lectr MVP.

- [ ] Environment setup guide
- [ ] How to integrate (baseURL swap)
- [ ] Known limitations
- [ ] New dev can integrate in <10 minutes

#### Task 11: E2E Validation
**Priority:** Low

Validate the MVP end-to-end with a real OpenAI client.

- [ ] Demo app using OpenAI SDK
- [ ] Switch baseURL to Lectr
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
| SHA-256 hashed API keys | Never store plaintext Lectr keys |
| `lc_` key prefix | Easy identification and validation |
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
