# Editorial Approval Tool — Implementation Plan

> Status: **Planning** · Owner: TBD · Target: internal/backoffice
>
> Replaces the BMM Excel sheet that Tobi, Johannes & co. use to go through what
> should be published from a stream. Editorial reviewers step through
> timestamps / chapters / markers for a Mediabanken asset and approve or reject
> each one for publishing.

## 1. Goal & scope

A simple tool where an editor:

1. **Creates a review session** tied to a Mediabanken asset.
2. **Imports markers** for that asset and can **add / edit / remove** rows.
3. **Approves or rejects** each marker for publishing (Ja / Nei toggle).
4. **Saves the session at any time** (persisted in SQLite).
5. When finished, **sends the approved set to Mediabanken** — _mocked for now_.

Two table view modes:

- **Enkel (Simple)** → read-only rows, only the Ja/Nei publish toggle is editable.
- **Rediger (Edit)** → every column is editable, rows can be added/removed.

Inline **video preview** per marker (Forhåndsvis / play button) using the existing
`getPreview` proxy, seeking to the marker's start time.

### Explicit non-goals for v1 (decided)

- **Playout integration is deferred.** Playout is an external system with an API
  and webhooks, but it is not integrated yet. The "Playout" selector and
  "import timestamps from Playout" in the original sketch are **postponed** until
  that integration exists. v1 imports markers from Mediabanken (Vidispine
  chapter metadata) and/or manual entry.
- **"Send to Mediabanken" is mocked.** The final publish step records that the
  session was sent but performs no real write-back / workflow yet.
- No real-time collaboration / multi-user locking. Last write wins.

## 2. Key decisions (resolved with product)

| Question                 | Decision                                                                     |
| ------------------------ | ---------------------------------------------------------------------------- |
| Playout timestamp source | Deferred — external API/webhooks not yet integrated. Not in v1.              |
| Marker source for v1     | Mediabanken asset chapters (Vidispine, like Export) + manual add/edit/remove |
| Unit of persistence      | A **session** (asset + its marker rows + publish/edit state)                 |
| Save behavior            | "Lagre" persists the session to SQLite; editable/re-openable later           |
| Final output             | "Send to Mediabanken" — **mocked** (marks session as sent, no real write)    |
| Storage engine           | **SQLite** (greenfield — no DB in the repo today)                            |

## 3. Architecture overview

Follows existing repo conventions exactly (see `CLAUDE.md`).

```
Frontend (Nuxt/Vue)                 Backend (Go / ConnectRPC)          Storage
──────────────────                  ─────────────────────────         ───────
/editorial/            ──────────▶  EditorialAPI handler       ──────▶ SQLite
  index.vue  (sessions list + new)    ListSessions                     editorial.db
/editorial/[id].vue    ──────────▶    CreateSession
  (marker table + preview)            GetSession
                                      SaveSession (upsert markers)
                                      DeleteSession
                                      SendToMediabanken (MOCK)
                                      ImportMarkers (Vidispine chapters)
                                    ▲
                                    │ Vidispine (chapters), getPreview (proxy)
```

The `/editorial/[id].vue` page **already exists as a 6-line stub** — we build it out.
`useTools`, `usePermissions`, and the auto-navigation wire the tool into the app
with no manual registration beyond the composable entries.

## 4. Data model (SQLite)

New DB file, path from an env var with a sensible default (mirrors how
`permissions.json` uses `CONFIG_ROOT`):

- `EDITORIAL_DB_PATH`, default `${CONFIG_ROOT}/editorial.db`.

**Driver:** `modernc.org/sqlite` (pure Go, no cgo) so builds/deploys stay simple.
Access via `database/sql` in a small `backend/editorial` package (flat, like
`bmm/`) — **DONE**
(no sqlc: not established in this repo, and the one non-trivial query is a
hand-orchestrated transaction anyway). Schema is created/migrated on startup with
an idempotent `CREATE TABLE IF NOT EXISTS`. DSN sets `foreign_keys(1)`,
`journal_mode(WAL)` and `busy_timeout(5000)`.

**Timestamps are stored as INTEGER epoch-millis** (not the SQL `TIMESTAMP` type
shown below) to stay independent of the driver's time-format handling; the store
converts to/from `time.Time` at the boundary.

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,          -- uuid
    vxid        TEXT NOT NULL,             -- Mediabanken asset id
    title       TEXT NOT NULL DEFAULT '',  -- display title (from asset or user)
    status      TEXT NOT NULL DEFAULT 'draft', -- draft | sent
    created_by  TEXT NOT NULL,             -- email
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    sent_at     TIMESTAMP                  -- set when "sent to Mediabanken" (mock)
);

CREATE TABLE IF NOT EXISTS markers (
    id           TEXT PRIMARY KEY,         -- uuid
    session_id   TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    sort_order   INTEGER NOT NULL,         -- explicit ordering for the table
    name         TEXT NOT NULL DEFAULT '', -- "Hvem eller hva" (person/thing)
    type         TEXT NOT NULL DEFAULT '', -- appell | vitnesbyrd | sang | ...
    start_ms     INTEGER NOT NULL,         -- marker start (ms)
    end_ms       INTEGER NOT NULL,         -- marker end (ms) → duration derived
    publish      INTEGER NOT NULL DEFAULT 0, -- 0/1 Ja/Nei
    source       TEXT NOT NULL DEFAULT 'manual', -- 'import' | 'manual'
    created_at   TIMESTAMP NOT NULL,
    updated_at   TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_markers_session ON markers(session_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_sessions_vxid   ON sessions(vxid);
```

Notes:

- **Duration** is derived (`end_ms - start_ms`), not stored.
- `type` is a free string in the DB but the UI offers a preset list (see §8) so
  new types don't require a migration.
- Timestamps stored in ms to avoid float drift; converted from Vidispine TC on import.

## 5. Permissions

New permission, added in the same way as `ShortsPermission`. **Two levels** (all
editorial users see all sessions — no per-user ownership):

- **`enabled`** → use the tool, see all sessions, accept/reject markers for
  publishing (the "Enkel" / simple view). No editing.
- **`admin`** → additionally add / remove / edit markers and sessions (the
  "Rediger" / edit view). Implies `enabled`.

### Proto (`api/v1/api.proto`) — DONE

```protobuf
message EditorialPermission {
  bool enabled = 1; // see all sessions + accept/reject
  bool admin   = 2; // add/remove/edit markers & sessions; implies enabled
}

message Permissions {
  // ... existing fields ...
  EditorialPermission editorial = 10;
}
```

### Backend helpers (`backend/api/v1/permissions.go`) — DONE

```go
// tool access: enabled OR admin
func (p *Permissions) CanEditorial() bool {
    return p.Admin || (p.Editorial != nil && (p.Editorial.Enabled || p.Editorial.Admin))
}
// edit rights: admin only
func (p *Permissions) CanEditorialEdit() bool {
    return p.Admin || (p.Editorial != nil && p.Editorial.Admin)
}
```

Mutating RPCs (create/save/delete/import markers) gate on `CanEditorialEdit`;
read + publish-toggle RPCs gate on `CanEditorial`.

### Default permissions (`backend/cmd/server/permissions.go`) — DONE

`Editorial: &apiv1.EditorialPermission{}` added to the default struct in
`PermissionsForEmail` (also backfilled the previously-missing `Shorts`).

### Frontend (`usePermissions.ts`) — DONE

```typescript
const canUseEditorial = cap(
  (m) => !!m.editorial && (m.editorial.enabled || m.editorial.admin),
);
const canEditEditorial = cap((m) => !!m.editorial?.admin);
```

Per memory: use `usePermissions()` everywhere — no inline `me.value?.editorial`.

### Admin editor (`AdminPermissionView.vue`) — DONE

Added an "Editorial" section with two switches (`enabled`, `admin`) so admins can
grant the permission via UI.

## 6. API surface (proto + RPCs)

Added to `APIService` in `api/v1/api.proto`; regenerate with `buf generate`.

```protobuf
// ── Messages ──────────────────────────────────────────────
message EditorialMarker {
  string id       = 1;
  int32  sort_order = 2;
  string name     = 3;
  string type     = 4;
  int64  start_ms = 5;
  int64  end_ms   = 6;
  bool   publish  = 7;
  string source   = 8; // "import" | "manual"
}

message EditorialSession {
  string id         = 1;
  string vxid       = 2;
  string title      = 3;
  string status     = 4; // "draft" | "sent"
  string created_by = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
  google.protobuf.Timestamp sent_at    = 8;
  repeated EditorialMarker markers = 9; // populated by GetSession only
}

message ListEditorialSessionsRequest {}                  // optionally filter later
message ListEditorialSessionsResponse { repeated EditorialSession sessions = 1; }

message CreateEditorialSessionRequest { string vxid = 1; string title = 2; }
message CreateEditorialSessionResponse { EditorialSession session = 1; }

message GetEditorialSessionRequest  { string id = 1; }
message GetEditorialSessionResponse { EditorialSession session = 1; }

// Full replace of the session's markers + title on save (simple & robust)
message SaveEditorialSessionRequest {
  string id = 1;
  string title = 2;
  repeated EditorialMarker markers = 3;
}
message SaveEditorialSessionResponse { EditorialSession session = 1; }

message DeleteEditorialSessionRequest { string id = 1; }

// Pull chapter markers from Vidispine for the session's asset (does NOT save;
// returns candidate rows for the client to merge into the table)
message ImportEditorialMarkersRequest  { string id = 1; }
message ImportEditorialMarkersResponse { repeated EditorialMarker markers = 1; }

// MOCK: mark the session as sent to Mediabanken
message SendEditorialToMediabankenRequest  { string id = 1; }
message SendEditorialToMediabankenResponse { EditorialSession session = 1; }

// ── Service methods ───────────────────────────────────────
rpc ListEditorialSessions(ListEditorialSessionsRequest) returns (ListEditorialSessionsResponse) {}
rpc CreateEditorialSession(CreateEditorialSessionRequest) returns (CreateEditorialSessionResponse) {}
rpc GetEditorialSession(GetEditorialSessionRequest) returns (GetEditorialSessionResponse) {}
rpc SaveEditorialSession(SaveEditorialSessionRequest) returns (SaveEditorialSessionResponse) {}
rpc DeleteEditorialSession(DeleteEditorialSessionRequest) returns (Void) {}
rpc ImportEditorialMarkers(ImportEditorialMarkersRequest) returns (ImportEditorialMarkersResponse) {}
rpc SendEditorialToMediabanken(SendEditorialToMediabankenRequest) returns (SendEditorialToMediabankenResponse) {}
```

**Save strategy:** `SaveEditorialSession` sends the whole marker list and the
backend does a transactional full replace (delete + insert within a tx). This is
the simplest correct model for a table editor and matches the "save whenever" UX.
Markers without an id are treated as new (backend assigns uuid).

Every RPC gates on `PermissionsForEmail(email).CanEditorial()` and, for
mutations, checks the session's `created_by` (or `admin`) before allowing edits.

## 7. Backend implementation

New files, following the `bmm/` support-package layout (flat, one package) plus
the `ShortsAPI` / `ExportAPI` handler pattern:

- `backend/editorial/store.go` — SQLite store, **package `editorial`** (open/migrate
  + CRUD, parameterized queries; markers replaced in a tx on save). **DONE.**
- `backend/editorial/store_test.go` — store unit tests against a temp DB. **DONE.**
- `backend/cmd/server/editorial.go` — `EditorialAPI` handler struct + RPCs.
  - Dependencies injected: `*editorial.Store` and the `vidispine.Client` (for import).
  - `ImportEditorialMarkers` reuses `vidispine.GetChapterMetaForClips` /
    chapter metadata (same source Export uses at `export.go:222`) and maps
    chapter title → `name`, TC → `start_ms/end_ms`.
- `backend/cmd/server/main.go` — construct the store, `NewEditorialAPI(store, vs)`,
  embed `EditorialAPI` in `ApiServer`.

Wiring checklist in `main.go`:

1. Open DB (`editorial.Open(dbPath)`), `defer store.Close()`.
2. `editorialAPI := NewEditorialAPI(editorialStore, vidispineClient)`.
3. Add `EditorialAPI` to the embedded `ApiServer` struct + literal.

## 8. Frontend implementation

### Routes

- `frontend/app/pages/editorial/index.vue` — **new**: session list + "New session"
  (pick a Mediabanken asset via `DesignSelect` / asset lookup, create → navigate to detail).
- `frontend/app/pages/editorial/[id].vue` — **build out the existing stub**: the
  marker table + inline preview (matches the sketch).

### `[id].vue` layout (mirrors the sketch)

- Top bar: asset title, **view-mode toggle** (Enkel / Rediger via `DesignSwitch`
  or two `DesignButton`s), **Importer** button (pull Vidispine markers),
  **Lagre** button (right-aligned), and **Send til Mediabanken** (mock).
- Left: **marker table**. There is **no `DesignTable`** in the library, so build a
  semantic `<table>` styled with Tailwind + Design primitives inside cells.
  Columns:
  | Hvem/hva (name) | Type | Varighet (duration) | Forhåndsvis (▶) | Publiseres? (Ja/Nei) |
  - **Enkel mode** (all editorial users): name/type/duration read-only; only the
    Ja/Nei `DesignSwitch` and the preview button are interactive.
  - **Rediger mode** (only users with `canEditEditorial`): name = `DesignInput`,
    type = `DesignSelect` (preset list below, allows free text), start/end
    editable (time inputs), add-row button, per-row remove button, drag or
    up/down to reorder (`sort_order`). Hide the mode toggle entirely for users
    without edit rights.
- Right: **video preview panel**. `<video :src>` from `api.getPreview({ VXID })`
  (same approach as `transcription/[id].vue:354`); clicking a row's ▶ seeks the
  element to that marker's `start_ms` and plays. No HLS lib needed (plain `<video>`).

Preset **types** (free-text still allowed): `appell`, `vitnesbyrd`, `sang`,
`tale`, `bønn`, `annet`. Kept client-side so adding one needs no migration.

### State & API

- `const api = useAPI()`; load via `GetEditorialSession`.
- Local reactive copy of markers; "Lagre" calls `SaveEditorialSession` with the
  full list. Track a dirty flag; warn on navigate-away if unsaved.
- Toast on save / send (`useToast` — note it was recently renamed; check the
  current composable name).
- `useAnalytics().page()` + `useHead({ title })` like other pages.

### Navigation (`useTools.ts`)

```typescript
{
    label: t("tools.editorial.title"),
    icon: "tabler:checklist",           // or tabler:clipboard-check
    description: t("tools.editorial.description"),
    to: "/editorial/",
    enabled: perms.canEditorial.value,
}
```

Tool then appears automatically on the home page and in the header.

## 9. i18n

Add an `editorial` block to both `frontend/locales/en.json` and `nb.json`
(Norwegian is the primary audience). Keys (nb values authoritative):

```
tools.editorial.title          "Publiseringsgjennomgang"
tools.editorial.description     "Gå gjennom og godkjenn markører for publisering"
editorial.newSession            "Ny gjennomgang"
editorial.selectAsset           "Velg Mediabanken-ressurs"
editorial.viewSimple            "Enkel"
editorial.viewEdit              "Rediger"
editorial.import                "Importer markører"
editorial.save                  "Lagre"
editorial.sendToMediabanken     "Send til Mediabanken"
editorial.col.name              "Hvem eller hva"
editorial.col.type              "Type"
editorial.col.duration          "Varighet"
editorial.col.preview           "Forhåndsvis"
editorial.col.publish           "Publiseres?"
editorial.publishYes            "Ja"
editorial.publishNo             "Nei"
editorial.addRow                "Legg til rad"
editorial.removeRow             "Fjern rad"
editorial.unsavedWarning        "Du har ulagrede endringer"
editorial.saved                 "Lagret"
editorial.sent                  "Sendt til Mediabanken"
editorial.status.draft          "Utkast"
editorial.status.sent           "Sendt"
```

(No HTML comments in the templates — per project convention.)

## 10. Build / phasing

1. **Proto + permissions** — add messages, RPCs, `EditorialPermission`; `buf generate`;
   `CanEditorial` helper; frontend `canEditorial`. _(Compiles, no behavior yet.)_
2. **Backend store** — `modernc.org/sqlite`, `go.mod`, store package + tests, migrations on boot. **DONE** (`backend/editorial/store/`, all tests pass).
3. **Backend handler** — `EditorialAPI` (CRUD + mock send), wire into `main.go`.
   Import from Vidispine chapters.
4. **Frontend list page** — `editorial/index.vue`: list sessions, create new.
5. **Frontend detail page** — build out `editorial/[id].vue`: table (both view
   modes), preview panel, save, import, mock send.
6. **Nav + i18n** — `useTools` entry, en/nb locales.
7. **Polish** — dirty-state warning, empty/loading states (`DesignSkeleton`),
   toasts, duration formatting (`useNumberFormat` if suitable).

## 11. Open questions / follow-ups

- **Playout integration** (future): once the external Playout API/webhooks exist,
  add a Playout stream selector and an import path; likely a new RPC
  `ImportEditorialMarkersFromPlayout`. Marker `source` field already allows this.
- **Real "Send to Mediabanken"** (future): replace the mock with the actual
  write-back — probably a Temporal workflow (like Shorts/Export) or a Vidispine
  metadata write. Decide the target field/shape then.
- **Session ownership / sharing**: RESOLVED — all editorial users see all
  sessions (no per-user filter on `ListEditorialSessions`). `created_by` is kept
  for display/audit only. Edit rights are governed by the `admin` flag, not
  ownership.
- **Asset picker UX**: reuse the Vault search for choosing the Mediabanken asset,
  or a plain VXID input? Vault search is nicer but heavier — confirm.
- **DesignTable**: none exists. If tables become common across tools, consider
  adding a `DesignTable` primitive rather than a one-off here.

```

```
