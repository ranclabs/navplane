# Lectr.ai MVP — Observability Proxy (Final Spec v1.1)

---

## 🚀 Overview

**One-line definition**

A zero-latency, OpenAI-compatible proxy that provides real-time, trustworthy observability of AI traffic without breaking streaming or reliability.

**Core Principle**

The request path is sacred. Nothing may degrade latency, correctness, or reliability.

---

## 🎯 MVP Goals

1. Integration takes < 2 minutes
2. Streaming is indistinguishable from OpenAI
3. No regression in latency or reliability
4. Observability is useful, honest, and clearly labeled
5. Infra cost is bounded
6. Data is isolated per org
7. System behaves predictably under failure
8. Trust boundaries are explicitly defined

---

## 🧱 MVP Scope

---

### 1) Transparent Proxy (Core)

**Supported Endpoint**

- `POST /v1/chat/completions`

**Provider Scope (Explicit)**

Lectr MVP supports OpenAI-compatible APIs only.

- All requests forwarded to OpenAI
- No multi-provider routing
- Schema is future-compatible with multi-provider

**Requirements**

- Full request/response passthrough
- Streaming (SSE) + non-streaming
- OpenAI SDK compatibility (Node + Python)
- No mutation of request or response

---

### 🔥 Non-Negotiable Behaviors

**Streaming correctness**

- Use `http.Flusher`
- Flush immediately per chunk
- No buffering
- Preserve chunk boundaries
- Target TTFT overhead < 10ms

**Context propagation**

- Cancel upstream request on client disconnect

```go
req = req.WithContext(r.Context())
```

**Resource safety**

- Always close response bodies
- No goroutine leaks
- No connection leaks

**Timeouts**

- Request timeout (60–120s)
- Upstream timeout
- Idle connection timeout

---

### 2) Org Key System (Isolation + Cost Control)

**Required Header**

```
X-Lectr-Key: lc_key_xxx
```

**Key Types**

| Key          | Purpose                   |
| ------------ | ------------------------- |
| `lc_key_xxx` | Org key (proxy usage) |

> Dashboard access is now handled by Auth0 — no separate dashboard token required.

**Key Lifecycle (MVP)**

**Create Org**

```
POST /v1/orgs
Authorization: Bearer LECTR_ADMIN_SECRET
Content-Type: application/json
```

Request body:

```json
{
  "name": "My Organization"
}
```

Response:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My Organization",
  "enabled": true,
  "org_key": "lc_key_xxx",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

`name` is required. `org_key` is only returned at creation time.

**Revoke Org Key**

```
POST /v1/orgs/{id}/rotate-key
Authorization: Bearer LECTR_ADMIN_SECRET
```

No request body. The org is identified by `{id}` in the path. Returns a new key; the old key is immediately invalidated.

Response:

```json
{
  "org_key": "lc_key_yyy"
}
```

**Management API Security (MVP)**

Management endpoints protected by a static admin secret.

- Stored as environment variable
- Required for all org lifecycle endpoints
- Temporary until full auth is introduced

---

### 3) Authentication (Dashboard)

Lectr uses **Auth0** for dashboard authentication.

**Flow**

1. User visits `dashboard.lectr.ai`
2. Auth0 handles login / signup / session
3. Dashboard scopes all data to authenticated user's orgs
4. No unauthenticated access permitted

**Supported login methods (MVP)**

- Email + password
- GitHub (important for developer audience)

**What this replaces**

The previous `dash_token_xxx` URL-based access model is removed entirely. Auth0 closes the security gap — dashboard URLs are no longer sensitive.

**Auth0 scope**

- Dashboard login and session management only
- Management API uses a separate admin secret (or Auth0 M2M tokens post-MVP)

**Known limitation**

Auth0 free tier has monthly active user limits. Monitor usage before scaling.

---

### 4) Security & Trust Model

Lectr is a transparent proxy and processes provider API keys in-memory.

**Guarantees**

- API keys are never persisted
- API keys are never logged
- API keys are never included in events
- API keys exist only in-memory per request

**Tradeoff (MVP)**

- Users must trust Lectr at runtime
- Secure key storage (BYOK) introduced in Phase 4

---

### 5) OpenAI API Key Handling

**Flow**

```
Client → sends OpenAI key → Proxy → forwards → OpenAI
```

**Hard Rules**

- ❌ Never stored
- ❌ Never logged
- ❌ Never persisted

---

### 6) Async Event Pipeline (Observability)

**Event Trigger**

- Emitted after request completes

**Event Includes**

- timestamp (captured at request completion, not DB insertion time)
- provider
- model_requested / model_actual
- latency (total + TTFB)
- status + error category
- streaming flag
- token count
- cost estimate
- org key

**Rules**

- ❌ No DB on hot path
- ✅ Buffered channel
- ✅ Batch writes
- ✅ Drop events under pressure

**Internal Metrics**

- `events_received`
- `events_written`
- `events_dropped`

**Failure Behavior**

| Condition     | Behavior            |
| ------------- | ------------------- |
| DB down       | events dropped      |
| worker slow   | events dropped      |
| proxy healthy | requests unaffected |

---

### 7) Observability Accuracy (Trust Rules)

**Token & Cost Strategy**

| Request Type  | Source           |
| ------------- | ---------------- |
| Non-streaming | Provider (exact) |
| Streaming     | Estimated        |

**Event Field**

```json
"token_source": "provider | estimated"
```

**UI Requirement**

Always display:

```
Estimated (±30% for streaming requests)
```

> Validate the ±30% figure against real traffic before publishing. It may be wider depending on model and content type.

---

### 8) Observability Dashboard (Read-Only)

**Access**

```
https://dashboard.lectr.ai
```

- Login required (Auth0)
- Data scoped to authenticated user's orgs
- No sensitive data in URLs

**Displays**

Usage

- Requests today
- Requests per minute
- Model distribution

Cost

- Total spend (estimated)
- Spend per model

Reliability

- Error rate
- Recent failures
- Latency avg + p95

**Degradation Indicator (Required)**

If events are dropped:

```
⚠️ Observability degraded — data may be incomplete
```

---

### 9) Cost Protection

- In-memory rate limiting per org key
- Max request size (1–2MB)
- Max concurrent streams
- Request timeouts
- Optional soft daily cap

**Known Limitation**

Rate limiting is per-instance only (not distributed).

---

### 10) Security

**TLS Enforcement**

- All endpoints must be HTTPS
- Plain HTTP not allowed

---

### 11) Privacy Guarantees

**Hard Rules**

- ❌ No prompts stored
- ❌ No responses stored
- ❌ No raw bodies logged

Only metadata is captured.

---

### 12) Health Endpoint

```
GET /health
```

Response:

```json
{
  "status": "ok",
  "worker": "ok",
  "event_queue_depth": 42
}
```

---

## 🏗️ Architecture

**Hot Path**

```
Client → Proxy → OpenAI → Proxy → Client
```

Constraints:

- No DB access
- No blocking operations

**Cold Path**

```
Proxy → Event Channel → Worker → Postgres
```

---

## 🧪 Definition of Done

**Proxy correctness**

- ✅ Streaming identical to OpenAI
- ✅ TTFT overhead < 10ms
- ✅ No request failures introduced

**Resource safety**

- ✅ No goroutine leaks
- ✅ No connection leaks
- ✅ Stable under high concurrency

**Observability**

- ✅ Dashboard updates within seconds
- ✅ Token source correctly labeled
- ✅ Degradation banner appears when needed

**Isolation**

- ✅ Org key required
- ✅ No cross-org data leakage

**Authentication**

- ✅ Auth0 login working (email + GitHub)
- ✅ Dashboard scoped to authenticated user
- ✅ No unauthenticated dashboard access

**Security**

- ✅ HTTPS enforced
- ✅ OpenAI key never stored or logged

**Cost protection**

- ✅ Rate limiting enforced
- ✅ Payload limits enforced
- ✅ Timeouts enforced

**Operations**

- ✅ `/health` endpoint functional
- ✅ Management API secured
- ✅ Key revocation works

---

## 📅 Execution Plan (Refined)

### Week 1 — Proxy Core

**Day 1–2**

- HTTP server
- OpenAI-compatible structs

**Day 3–4**

- Forwarding logic
- Header passthrough
- Context propagation

**Day 5–6 (Critical)**

- Streaming passthrough
- Flushing correctness
- TTFT measurement

**Day 7**

- Org key validation
- Rate limiting
- Request size limits

---

### Week 2 — Observability + Auth

**Day 8–9**

- Event schema
- Async channel
- Begin load testing

**Day 10–11**

- Worker + batch DB writes
- Failure simulation (DB down, slow worker)

**Day 12–13**

- Auth0 integration
- Dashboard UI
- Charts (usage, cost, latency)
- Data scoping to authenticated user

**Day 14**

- Final polish
- Streaming validation (real SDKs)
- Leak detection
- Failure scenario testing

---

## 📘 Runbook (MVP)

**DB Down**

- Events dropped
- Dashboard incomplete
- Proxy unaffected

**Worker Overloaded**

- Events dropped
- Degradation banner shown

**Proxy Restart**

- In-memory events lost
- No persistence guarantee

**High Traffic**

- Rate limiting triggers
- Requests may return 429

**Auth0 Down**

- Dashboard inaccessible
- Proxy unaffected (Auth0 is not on the hot path)

---

## 📈 Success Criteria

You have succeeded if:

- Teams route production traffic through Lectr
- Streaming UX is flawless
- Dashboard is trusted
- Infra cost is controlled
- Users say: _"This just works."_

---

## 🧭 Post-MVP Roadmap

**Phase 2 — Insight**

- Better cost accuracy
- Anomaly detection

**Phase 3 — Simulation**

- Shadow routing
- Cost comparison

**Phase 4 — Control**

- BYOK
- Fallback routing
- Budgets
- Auth0 M2M tokens for management API

---

## 🔥 Final Principle

We earn the right to control AI traffic only after we prove we can observe it perfectly.
