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
**Branch:** `ronald/02-org-model` (in review)

- [x] Create org package with manager/datastore pattern
- [x] Implement `Org` model (ID, Name, APIKeyHash, Enabled, timestamps)
- [x] Implement API key generation (`np_<uuid>` format)
- [x] Implement SHA-256 key hashing for storage
- [x] Create auth middleware (Bearer token extraction)
- [x] Wire auth middleware to `/v1/chat/completions`
- [x] Add comprehensive tests with sqlmock
- [x] Update AGENTS.md with manager/datastore guidelines

---

### In Progress

_(None currently)_

---

### Pending

#### Task 3: Org-Level Kill Switch
**Priority:** High

Add the ability to instantly disable all traffic for an organization.

- [ ] Add `enabled` flag check in auth middleware
- [ ] Admin endpoint to toggle org status (`PUT /admin/orgs/:id/enabled`)
- [ ] Disabled org requests fail immediately with 403
- [ ] No provider calls made for disabled orgs
- [ ] Re-enabling restores traffic instantly

#### Task 4: BYOK Vault – Secure Provider Key Storage
**Priority:** High

Implement secure storage for provider API keys using encryption.

- [ ] Postgres schema for provider keys (per-org)
- [ ] Envelope encryption (master key + DEK)
- [ ] Encrypt on write, decrypt only in memory
- [ ] Redact sensitive fields from logs
- [ ] No plaintext keys in DB or logs

#### Task 5: Header Swapping (NavPlane Key → Provider Key)
**Priority:** High

Swap NavPlane API keys with provider API keys during request forwarding.

- [ ] Extract org from request context
- [ ] Load provider key from vault
- [ ] Inject correct provider `Authorization` header
- [ ] Remove NavPlane auth headers before forwarding
- [ ] Fail immediately if key is missing or deleted

#### Task 6: Reverse Proxy Director
**Priority:** Medium

Create a Director function that rewrites outbound requests before forwarding.

- [ ] Rewrite request URL to provider endpoint
- [ ] Strip NavPlane auth headers
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

#### Task 8: Minimal Admin API
**Priority:** Medium

Expose admin endpoints used by the dashboard.

- [ ] Create/delete provider keys
- [ ] Enable/disable org
- [ ] List recent requests
- [ ] Auth-protected endpoints
- [ ] Changes take effect immediately

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
| SHA-256 hashed API keys | Never store plaintext keys |
| `np_` key prefix | Easy identification and validation |
| PostgreSQL triggers for `updated_at` | Auto-update timestamps on modifications |
| No exposed DB port | Security best practice |
| sqlmock for testing | Unit test DB operations without real database |
