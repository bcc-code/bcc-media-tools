# Transcription Tool — Bugs & Improvements

> Status: **Phase 0 done** (§7) · **Phase 2 in progress** (§9) · Owner: TBD
>
> Tracking note for fixing the transcription editor (`/transcription/[id]`)
> after several user reports in Oct 2025 that edits are lost, rows appear
> duplicated, and it is hard to add missing text.

**Scope — this is a correction tool, not a transcription tool.** The text is
already produced by automatic speech recognition. The users are volunteers doing
a proofreading pass: fixing a misheard word, deleting something the ASR
hallucinated, and filling in the occasional phrase it dropped. They should
almost never be typing a whole sentence, and never a whole segment. Every design
decision below follows from that: **the cheapest possible small edit wins over
any authoring feature**, and unchanged words must keep their original timings —
the timings are the part of the data the user cannot re-create.

## 1. User reports (verbatim)

| #   | Report (nb)                                                                                                                                                                | Diagnosis                                                                                     |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| R1  | "Når jeg nå er ferdig og refresher siden, kommer mange av radene dobbelt. Altså både en originale raden og den redigerte."                                                 | [B1](#b1) + [B7](#b7)                                                                         |
| R2  | "Jeg opplever at jeg retter teksten, men når jeg går tilbake, så har det falt tilbake til slik det var før. Lagrer den ikke underveis?"                                    | [B1](#b1)                                                                                     |
| R3  | "Hvis jeg scroller opp for å se det jeg tidligere har rettet på, så har det hoppet tilbake til original tekst."                                                            | [B1](#b1) — exact signature                                                                   |
| R4  | "Det er vanskelig å legge til tekst der teksten helt er uteblitt. Det kommer opp + tegn innimellom, men ikke etter hver kolonne, og da må jeg fylle alt inn i en kolonne." | [B5](#b5), [B6](#b6)                                                                          |
| R5  | "Muligheten til å legge inn en ekstra rad mellom hver rad. Det er bare mulig mellom noen av radene."                                                                       | [B5](#b5)                                                                                     |
| R6  | "Nå er den helt tilbake til originalen selv om jeg laster inn siden på nytt."                                                                                              | [B2](#b2) + [B3](#b3)                                                                         |
| R7  | "[person 1] sin tale er lagret og riktig, men ingenting av [person 2] tale lar seg lagre. Føles som jeg har brukt opp muligheten til å endre 😅"                           | [B3](#b3) (draft deleted + access revoked), possibly [B4](#b4)                                |
| R8  | "Det fungerte når jeg refreshet svært ofte. Noen ganger måtte jeg gjøre det dobbelt. Gjorde det ca hvert 5 min."                                                           | Confirms [B1](#b1): a page reload is the only thing that re-syncs the two copies of the state |

R8 is the smoking gun. A full page reload is the _only_ code path that makes the
edited data and the rendered data point at the same array again — see below.

## 2. Root cause: two copies of the same state that drift apart

The editor keeps the transcription in **two places that are supposed to be the
same array but stop being so on the first edit**:

- `transcription` — the whole result object; `transcription.segments` is **what
  gets rendered** (`TranscriptionEditor.vue:93-102` feeds it to `useVirtualList`)
  and **what gets submitted** (`[id].vue:88`).
- `segments` — the `v-model` array; **what gets edited** and **what gets written
  to localStorage** (`[id].vue:195-202`).

`setTranscription()` (`[id].vue:76-80`) aliases them (`segments.value =
transcription.value.segments`), so they start out identical. Then:

```
handleSegmentUpdate()   TranscriptionEditor.vue:33-37
  segments.value = arr          // new array of copies → model updated
                                // transcription.segments STILL the old array
```

Only `setSegments()` (`[id].vue:220-224`) writes back into
`transcription.segments`, and it is called from exactly one place:
`addNewSegmentAt` (the `+` button). So after the first text edit the two diverge
and stay diverged until the page is reloaded.

```
         edit text          add segment (+)        reload page
 ─────────────────────────────────────────────────────────────────
 segments        arr2            arr3                 saved
 transcription   arr1 (stale)    arr3  ← re-synced    saved  ← re-synced
 rendered from   arr1            arr3                 saved
 submitted       arr1  ✗         arr3  ✓              saved  ✓
```

### <a id="b1"></a>B1 — Edits revert on scroll, and are lost on submit · **critical** · ✅ fixed

Consequences of the divergence above:

1. **Scrolling reverts text (R3).** The list is virtualised
   (`TranscriptionEditor.vue:96-102`), so a row that scrolls out of the window is
   unmounted and re-created from `transcription.segments` — the stale array. The
   typed text only survived in the `contenteditable` DOM (Vue never re-rendered
   it, because from its point of view the text never changed), so the remount
   throws it away. `TranscriptionSegmentEditor.vue:18` makes this worse: the
   local `words` ref is seeded from props exactly once and never re-synced.
2. **Submit uploads the original (R2, R6).** `submitToMediabanken` sends
   `transcription.value` (`[id].vue:88`), i.e. the stale array. Every edit made
   after the last page load — unless the user happened to press `+` afterwards —
   is silently dropped.
3. **Deleting a row wipes every text edit.** `handleSegmentToggleDelete`
   (`TranscriptionEditor.vue:27-30`) rebuilds the model by filtering
   `props.transcription.segments` — the stale, unedited array.
4. **Index desync.** `handleSegmentUpdate(s.index, …)` uses an index into the
   _rendered_ array to write into the _model_ array. Once the two have different
   lengths (after a delete), an edit lands on the wrong segment — you get the
   same text on a row you never touched while the row you did touch still holds
   the original. That is a plausible source of R1's "both the original row and
   the edited row".

**Fix:** one source of truth. Drop `transcription.segments` entirely as a render
source; render from the `segments` model, keep `text` derived. Ideally replace
the ad-hoc refs with a `useTranscriptionDraft(vxid)` composable that owns
load / edit / delete / insert / persist and exposes immutable updates keyed by a
stable segment id rather than array index.

### <a id="b2"></a>B2 — After submit there is no way to see or continue your work · **critical**

`SubmitTranscription` (`backend/cmd/server/transcription.go:121`) starts the
`ImportSubtitles` Temporal workflow and returns immediately. The workflow writes
the edited JSON back as the `transcription_json` shape with `Replace: true`, and
`GetTranscriptionJSON` reads the first format named `transcription_json` — so a
reload _should_ show the edited version once the workflow finishes. But:

- the frontend does not wait for or poll the workflow, and
- on submit the volunteer is removed from the `_AccessibleByTools` collection
  (`transcription.go:52-67`), so `GetTranscriptionPreview` starts returning
  `PermissionDenied` and `onMounted` bails out at `[id].vue:127` before even
  loading the transcription.

So the user is told "Transkripsjon er sendt til Mediabanken", loses access, and
if anything went wrong downstream they see the original text with no recourse —
"føles som jeg har brukt opp muligheten til å endre" (R7) is literally what
happens.

### <a id="b3"></a>B3 — The local draft is deleted before the work is known to be safe · **critical** · ✅ fixed

`[id].vue:90` — `localStorage.removeItem(key)` runs as soon as the workflow has
been _started_. If the Vidispine import job fails, or the shape is not actually
replaced, the only copy of the user's hours of work is gone. Combined with B2
the user cannot even get back in to redo it.

**Fix:** keep the draft until the workflow is confirmed complete (submit should
return a workflow id the frontend can poll, or the handler should wait), and
keep read access to submitted items.

### <a id="b4"></a>B4 — localStorage can silently fail to save · **high** · ✅ fixed

`[id].vue:195-202` serialises the **entire** transcription — including the
`tokens` array on every segment — on every keystroke-driven update. For a long
talk this is easily several MB and can exceed the ~5 MB origin quota. The write
is unguarded, so `QuotaExceededError` propagates out of the watcher and **no
edits are ever persisted, with no error shown to the user**. This fits R7
exactly: one talk saves fine, another one never saves at all.

**Fix:** strip `tokens` (and the other whisper metadata) from the draft, debounce
the write, wrap it in `try/catch`, surface a visible "kunne ikke lagre" state,
and move to IndexedDB or — better — server-side drafts ([I1](#i1)).

Related: the error path in `reset()` (`[id].vue:70-71`) sets `segments.value =
[]`, which triggers the same watcher and **overwrites a good draft with an empty
one**. A transient network blip during "Tilbakestill" destroys the draft.

### <a id="b5"></a>B5 — You can only insert a row where there is a ≥1 s gap · **high** · ✅ fixed

`canAddSegment` (`TranscriptionEditor.vue:54-60`) only shows the `+` when
`next.start >= curr.end + 1`. That is exactly R4/R5: the `+` appears "innimellom"
and nowhere else, so users cram missing sentences into the neighbouring row.
`addNewSegmentAt` (`:66`) additionally returns early when there is no previous or
next segment, so you can never insert **before the first** or **after the last**
row.

**Fix:** allow insertion between any two rows (and at both ends); derive the new
row's timing by splitting the neighbour's interval when there is no gap, and let
the user adjust start/end.

### <a id="b6"></a>B6 — One `contenteditable` per word blocks the three edits this tool exists for · **high** · ✅ fixed

`TranscriptionSegmentEditor.vue:67-79` renders each word as its own
`contenteditable` span, which is a reasonable shape for _adjusting timings_ and
the wrong shape for _correcting text_. The three operations that make up
essentially the whole job are each awkward or impossible:

| Correction                                     | Today                                                                                                                                                                                         |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Fix a word** ("Kåre" → "Kaare")              | Works, but it is the only thing that does.                                                                                                                                                    |
| **Delete a word** (ASR hallucinated it)        | Emptying the span leaves a zero-width box that still exists in the data and that you can never click into again — R4's "teksten helt er uteblitt". There is no way to remove the word itself. |
| **Add a missing word/phrase** (ASR dropped it) | No way to create a word. You must type it into a neighbouring word's box, which merges two words into one token and destroys that word's timing.                                              |

On top of that there is no select-across-words, no working undo (⌘Z is scoped to
one span), and `handleTextUpdate` reads `target.innerText`, so a paste or a stray
Enter injects newlines straight into the subtitle text.

**Fix:** make the whole segment one editable field, and re-derive word timings on
change by aligning the new token sequence against the old one. Because this is a
correction pass the alignment is cheap and near-exact: the vast majority of
tokens are unchanged and keep their original start/end verbatim; only inserted
runs need timings, interpolated across the span between their surviving
neighbours. A word-diff (LCS over tokens) is enough — no re-alignment against the
audio. Keep the per-word boxes only as an optional "juster tider" mode for the
rare case where a timing is actually wrong.

### <a id="b7"></a>B7 — The virtual list is misconfigured and rows visibly duplicate/overlap · **high**

`useVirtualList(..., { itemHeight: 80 })` (`TranscriptionEditor.vue:96-102`)
assumes every row is exactly 80 px, but rows are `min-height: 80px` (`:142`) at
`text-xl` and wrap to two or three lines, and the `+` buttons (`:149-161`) are
extra children the virtualiser does not know about. The window offset therefore
drifts from the real content height, so rows overlap and the same content can be
painted at more than one scroll offset. On top of that the `TransitionGroup`
(`:113-124`) gives leaving rows `absolute` positioning for 300 ms while the
virtualiser recycles rows on every scroll event, which leaves ghost rows stacked
on top of live ones.

This is the most likely source of R1's "mange av radene dobbelt" as a _rendering_
artefact (B1.4 explains the data-level variant). **Needs a reproduction to
confirm which of the two dominates.**

**Fix:** use dynamic item sizing (`useVirtualList`'s `itemHeight` callback, or
`@tanstack/vue-virtual`), take the `+` rows out of the virtualised item flow, and
drop the `TransitionGroup` inside the virtual window.

### <a id="b8"></a>B8 — `SubmitTranscription` has no authorization check at all · **high (security)** · ✅ fixed

`backend/cmd/server/transcription.go:121-142` never calls `getEmail` or
`PermissionsForEmail`. Any request that reaches the backend can overwrite the
subtitles of any VXID. `GetTranscriptionPreview` right above it does the full
check; `GetTranscription` (`:68`) checks the tool permission but — unlike the
preview endpoint — not `inTranscriptionCollection`, so any Mediabanken-level user
can read any transcription.

**Fix:** mirror the `GetTranscriptionPreview` checks in both handlers.

### <a id="b9"></a>B9 — `clearLocalData` wipes localStorage for the whole origin · **medium** · ✅ fixed

`[id].vue:107-110` calls `localStorage.clear()`, destroying **every other
transcription draft** plus all user settings (`seekOnFocus`, `splitterSize`,
`hasOpenedManual`, …). It is also reachable by hovering the top-left text and
pressing `c` — which fires while typing if the pointer happens to rest there
(`:112-118`).

**Fix:** remove only this document's key; drop the hidden hotkey or put it behind
an explicit confirm.

### <a id="b10"></a>B10 — Video-follow crashes / skips the first segment · **medium**

`[id].vue:141-175`:

- `if (!index) return` (`:158`) treats index `0` as "not found", so the first
  segment never gets highlighted or scrolled to.
- `segmentelements.value[index].$el` (`:163-165`) assumes the row is mounted. It
  is a virtual list, so any segment outside the render window is missing from the
  map and this throws a `TypeError` inside `ontimeupdate` — i.e. on every frame
  once playback runs past the visible window.
- `segmentelements` is keyed by absolute index and never cleaned up, so it
  accumulates references to unmounted components whose `$el` is detached;
  `scrollIntoView` on those silently does nothing.

### <a id="b11"></a>B11 — Smaller correctness issues · **low**

- `fileName` is set to `"ts-" + VXID` (`[id].vue:22,132`) and
  `downloadTranscriptionJSON` strips everything after the last `.`
  (`utils/transcription.ts:107`). Since that name has no dot, the download is
  named **`-edited.json`**.
- `transcription.text` is never recomputed from the edited segments, so the
  `text` field submitted to Mediabanken (`transcription.go:165`) is always the
  original full text.
- The frontend `Segment` type (`utils/transcription.ts:1-14`) uses
  `avg_logprob` / `compression_ration` (sic) / `no_speech_prob`, but the
  generated protobuf type uses `avgLogprob` / `compressionRatio` /
  `noSpeechProb`. The hand-written type does not describe the objects actually in
  flight, and the fields `addNewSegmentAt` sets (`TranscriptionEditor.vue:73-79`)
  are dropped on submit.
- `deletedIndexes` (`TranscriptionEditor.vue:17`) is index-based and not
  persisted, so it points at the wrong rows after any insert or after a reload.
- New segments get `id: (prev.id + next.id) / 2` — a non-integer id that the
  backend then renumbers anyway (`transcription.go:150`).
- The downloader toast string is hard-coded English
  (`TranscriptionDownloader.vue:24`).
- `TranscriptionSegmentEditor` declares `addBefore` / `addAfter` emits that are
  never used.

## 3. UX & flow improvements

### <a id="i1"></a>I1 — Server-side drafts with autosave (the actual ask)

"Lagrer den ikke underveis eller er det noen måte å lagre på?" (R2). Today the
only persistence is localStorage, and the only button says **"Send til
Mediabanken"** — which is final and revokes access. Users have no model of when
their work is safe.

Proposal: a `SaveTranscriptionDraft` / `GetTranscriptionDraft` RPC pair backed by
the same SQLite store the editorial tool is introducing
(`docs/editorial-approval-tool.md`), autosaved on a debounce, with an explicit
"Lagret kl. 14:32" indicator in the header (the editorial tool already has a
saving indicator — reuse it). Drafts survive a different browser/machine and can
be recovered by an admin.

### I2 — Make the save/submit distinction obvious

- "Lagre (kladd)" vs "Send inn (ferdig)" as two clearly different actions.
- Move the destructive **"Tilbakestill"** away from the saved-state text; require
  a confirm.
- Say what submit actually does: _"Dette laster opp undertekstene og du mister
  tilgang til filen."_ The confirmation text already says this — the button that
  opens it is labelled `transcription.save` / "Send til Mediabanken", which reads
  like a save.
- Show workflow progress and a real success state after submit instead of an
  optimistic toast and an immediate `navigateTo("/transcription")`.

### I3 — Editing model, ordered by how often a corrector needs it

Ranked for a proofreading pass, not for authoring:

1. **Segment-level editing** with timings re-derived by token alignment
   ([B6](#b6)). Unlocks delete-a-word and add-a-word, which are two thirds of
   the job.
2. **Undo/redo across the document** (⌘Z). The single highest-value affordance
   in a correction tool: it makes every edit safe to attempt, which is exactly
   what these users currently do not feel.
3. **Find & replace.** ASR gets the same proper noun wrong every time — a name,
   a place, a term of art. One replace-all can be a third of the corrections in
   a talk. Today each of the ~40 occurrences is fixed by hand.
4. **Insert a row anywhere** ([B5](#b5)), including before the first and after
   the last, for speech the ASR dropped entirely.
5. **Split at the caret / merge with previous.** ASR mis-segments across
   sentence boundaries fairly often; both are one keystroke if segments are
   editable as text.
6. **Editable start/end times per segment** — only really needed for rows the
   user inserted. `secondsFromFormattedTime` already exists in
   `utils/transcription.ts` and is unused. Low priority.

Explicitly **not** worth building: rich text, speaker labels, styling,
multi-track, or anything that treats this as a writing surface.

### I4 — Navigation & orientation

- **Surface `confidence`.** Every word already carries one and the UI throws it
  away. Shading low-confidence words, and a "hopp til neste usikre ord"
  shortcut, turns a linear read-the-whole-thing pass into a targeted one — the
  biggest single reduction in how much work a correction actually is.
- Progress indicator: how many segments reviewed / left.
- Keyboard: play/pause, ±3 s, and play-current-segment without leaving the
  editor; the manual (`TranscriptionManual.vue`) currently lists only ↑ ↓ Tab.
- Keep the video sticky when scrolling; make "Slettemodus" less modal (a red
  ring around the whole app is alarming for what is a per-row action that already
  has a trash button).

### I5 — Onboarding & trust

- A short intro (the manual only auto-opens once, after 1 s, and only lists three
  shortcuts) explaining: your changes are saved automatically, submit is final,
  and — importantly — **you are correcting a machine transcription, not writing
  one**. Say that you are not expected to catch everything, and that skipping a
  segment you are unsure about is fine. Several reports read like people who
  thought they had broken something.
- Norwegian-first copy — the users are volunteers, not staff.

## 4. Suggested order of work

| Phase                        | Scope                                                                                | Why first                                                                                             |
| ---------------------------- | ------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------- |
| **0 — stop the bleeding** ✅ | [B1](#b1), [B3](#b3), [B4](#b4) (guard + strip `tokens` + debounce), [B9](#b9)       | These lose work. Nothing else matters until edits reliably survive.                                   |
| **1 — trust**                | [I1](#i1) server drafts, [I2](#i2) save/submit clarity, [B2](#b2) post-submit access | Users can see their work is safe.                                                                     |
| **2 — editing**              | [B6](#b6), [B5](#b5), [I3](#i3) 1-3, [I4](#i4) confidence                            | Delete/add a word become possible at all; undo, replace-all and confidence shading cut the work down. |
| **3 — rendering**            | [B7](#b7), [B10](#b10)                                                               | Removes the remaining "weird" behaviour.                                                              |
| **4 — polish**               | [B8](#b8) (do earlier if exposure warrants), [B11](#b11), [I4](#i4), [I5](#i5)       |                                                                                                       |

## 5. Open questions

- **Open, and worth 5 minutes:** is the `transcription_json` shape actually
  replaced on re-import, or can an asset end up with two `transcription_json`
  formats? `GetTranscriptionJSON` picks the first match, so a second format
  would explain R6 on its own — and that would be a backend bug phase 0 did not
  touch. → look at a submitted asset's formats in Cantemo.
- ~~Should submitted transcriptions stay readable to the volunteer?~~
  **Answered (2026-09-24): no.** If a transcription needs more work it is simply
  shared again. Current behaviour stands.
- ~~Do we want a review step (volunteer submits → staff approves)?~~
  **Answered (2026-09-24): not for now.**
- Reproduce R1 on a real asset to confirm whether the doubling is [B7](#b7)
  (rendering) or [B1](#b1).4 (data).

## 6. Files involved

```
frontend/app/pages/transcription/[id].vue                     state, persistence, submit
frontend/app/pages/transcription/index.vue                    local-JSON entry point
frontend/app/components/transcription/TranscriptionEditor.vue segment list, virtualisation, insert/delete
frontend/app/components/transcription/TranscriptionSegmentEditor.vue  per-word contenteditable
frontend/app/components/transcription/TranscriptionDownloader.vue
frontend/app/components/transcription/TranscriptionManual.vue
frontend/app/utils/transcription.ts                           types, SRT/JSON export
backend/cmd/server/transcription.go                           Get/Submit/Preview handlers
api/v1/api.proto                                              Transcription messages + RPCs
```

Upstream: `bcc-media-flows` `workflows/ingest/import_subtitles.go` (`ImportSubtitles`),
`services/cantemo/client.go` (`GetTranscriptionJSON`, `GetACL`).

## 7. Phase 0 — what was changed

Edits now survive scrolling, deleting and submitting. One document, one array.

**New**

- `frontend/app/utils/transcriptionDraft.ts` — draft encode/decode and storage.
  Storage is injected and `writeDraft` returns a result instead of throwing, so
  a full quota is reported rather than swallowed.
- `frontend/app/composables/useTranscriptionDraft.ts` — owns the document for
  one asset: load, autosave, reset, submit. The API client and the storage are
  injectable.

**Changed**

- `utils/transcription.ts` — segments carry a `uid` and a `deleted` flag.
  Added pure operations (`updateSegment`, `toggleSegmentDeleted`, `setWordText`,
  `canInsertAfter`, `insertSegmentAfter`, `toTranscription`) so the editing
  rules are testable without a component.
- `TranscriptionEditor.vue` — renders from the `v-model` instead of
  `transcription.segments`; rows are keyed and addressed by `uid`. Deleting
  marks a row rather than filtering a second array, which is what discarded
  text edits.
- `TranscriptionSegmentEditor.vue` — dropped the local `words` copy that was
  seeded from props once and never re-synced. The contenteditable is now
  explicitly uncontrolled via a small directive that writes only when the value
  really differs, so the caret survives typing and a re-created row still shows
  current text.
- `pages/transcription/[id].vue` — state moved into the composable; the page is
  layout, the video and the submit dialog. Shows the last save time, and an
  error banner if saving fails.
- `pages/transcription/index.vue` — adapted to the same model.

**Behaviour**

- Submitting no longer deletes the local draft, and records `submittedAt` on it.
- Drafts are written debounced (500ms), without `tokens`, and never as an empty
  document over a good one.
- A failed "Tilbakestill" leaves the current work alone.
- `localStorage.clear()` and the hidden hover-plus-`c` shortcut are gone.
- Segments whose text ends up empty are dropped on submit/download instead of
  becoming blank cues — which is also how a corrector removes invented speech,
  given that words cannot be deleted individually yet ([B6](#b6)).

**Deliberately not touched** — [B5](#b5) (the ≥1s gap rule still limits where
`+` appears), [B6](#b6), [B7](#b7) (fixed-height virtual list), [B10](#b10),
[B11](#b11) metadata field names.

## 8. Tests

`pnpm test` (vitest, Nuxt environment — `frontend/vitest.config.ts`). 68 tests
in `frontend/test/`, covering the extracted logic rather than the components:

| File                            | Covers                                                                                                             |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `transcription.spec.ts`         | segment operations — uid addressing, delete marking, word edits, insert rules, and what `toTranscription` lets out |
| `transcriptionDraft.spec.ts`    | draft encode/decode, token stripping, quota and unusable-storage handling, per-asset key isolation                 |
| `useTranscriptionDraft.spec.ts` | load / autosave / reset / submit, with the API and storage injected                                                |

The tests were checked against deliberately reintroduced regressions — every one
of these is caught:

- `toTranscription` keeping deleted rows, or not re-deriving text from words
- `updateSegment` failing to match (the stale-array behaviour)
- the draft keeping `tokens`
- a quota error escaping instead of being reported
- an empty document overwriting a good draft
- a failed reset wiping the work
- submit deleting the local draft

**Still not verified.** The three scenarios in §4 have not been exercised
against a real asset in a browser, and nothing covers the component layer —
in particular the uncontrolled-contenteditable directive and the virtual list.

## 9. Phase 2 — in progress

Ordering note: phase 2 is being done before phase 1. Phase 0 made local
persistence reliable, so the acute data-loss pain is gone; what remains in the
users' own words is editing friction, which needs no backend. Phase 1's
centrepiece (I1, server-side drafts) is better built alongside the editorial
tool's SQLite work.

### B8 — authorization ✅

`backend/cmd/server/transcription.go`. `SubmitTranscription` had no check at
all; `GetTranscription` checked the tool permission but not whether the asset
was shared with the caller, and passed HTTP statuses where ConnectRPC expects
its own codes (which is why frontend errors arrived tagged `[unknown]`).

All three handlers now go through one policy, `transcriptionAccess`, which takes
the permissions and a `sharedForEditing` func. Splitting it that way keeps the
Cantemo ACL lookup out of the decision, so the policy is unit-tested
(`cmd/server/transcription_test.go`) and the lookup is skipped for admins who
would pass regardless.

### B6 — segment-level editing ✅

Each segment is now one editable field instead of one `contenteditable` per
word, so adding, deleting, splitting and merging words are ordinary typing.

Word timings are re-derived on every edit by `realignWords`
(`utils/transcription.ts`): an LCS diff over the tokens, matched on a
case- and punctuation-insensitive key. Words that survive keep their original
start/end exactly; a run of new tokens takes over the span of the words it
replaced, or the pause between its surviving neighbours if it is a pure
insertion. Nothing is re-aligned against the audio — in a correction pass
almost every token survives, so the diff carries the timings.

Known limitation: inserting between two words with no pause between them gives
the new word a zero-length span. It only affects word-level SRT export; the
segment-level SRT and the JSON are unaffected.

The field is uncontrolled while focused. The caret would otherwise be reset on
every keystroke, both by Vue patching the text node and by the canonical text
being whitespace-normalised. It re-syncs on blur and whenever the row is
re-created. Enter is suppressed (a cue is one line; split-at-caret is [I3](#i3))
and paste is forced to plain text via the Selection/Range API.

Because that paste is a programmatic DOM edit rather than
`document.execCommand` (deprecated), it is outside the browser's native undo
stack. Not a loss against the plan: native undo cannot span segments anyway, so
[I3](#i3)'s document-level undo has to capture paste explicitly regardless.

Seek-on-focus survives the change: the caret position is mapped back to a word
via `wordAtOffset`, so clicking into a word still seeks the video there.

### B5 — insert a row anywhere ✅

`insertSegmentAt` replaces `canInsertAfter`/`insertSegmentAfter`. Index `-1`
inserts before the first row, and there is no longer a gap requirement. Where
there is a pause the new row fills it; where there is none it borrows up to 2s
from the following row, leaving that row at least 0.4s, and refuses to borrow
from a row too short to give. A row appended at the end is clamped to the video
duration, which the page now passes down from the video element.

The `+` is no longer conditional. It sits on the row boundary and appears on
hover, which also removes the "why is there no + here" question entirely.

### I2 — submit reads as submit ✅

The button key was `transcription.save`. Renamed to `transcription.submit`, and
**"Tilbakestill" now asks for confirmation** — it sits next to the save
indicator and discarded every correction in one click.

The manual's shortcut list was updated: tab moves between segments now, not
between words.

### Still to do in phase 2

- [ ] Nothing — see §10 for what is left overall.

## 10. What is left

| Item                 | Why it is still open                                                                                                                                                                      |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [B7](#b7)            | Fixed-height virtual list against variable-height rows. Phase 0's layout fix bounded the scroll container, so virtualisation now actually windows, but `itemHeight: 80` is still a guess. |
| [B10](#b10)          | Video-follow skips the first segment; `segmentelements` is never cleaned up.                                                                                                              |
| [B11](#b11)          | The `avg_logprob` / `compression_ration` field-name mismatch between the hand-written type and the generated protobuf.                                                                    |
| [I1](#i1)            | Server-side drafts — the one thing that still makes a browser the only place the work lives. Best built with the editorial tool's SQLite work.                                            |
| [B2](#b2)            | Submit is still optimistic: it reports success when the workflow _starts_. Needs the workflow id back from the API.                                                                       |
| [I3](#i3)            | Undo/redo and find & replace — the two highest-value editing affordances still missing.                                                                                                   |
| [I4](#i4), [I5](#i5) | Confidence shading, progress, onboarding copy.                                                                                                                                            |

The `transcription_json` question in §5 is still unanswered and is the cheapest
thing on this list to resolve.
