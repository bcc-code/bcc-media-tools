# Transcription Tool — Bugs & Improvements

> Status: **Analysis done, nothing fixed yet** · Owner: TBD
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

### <a id="b1"></a>B1 — Edits revert on scroll, and are lost on submit · **critical**

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

### <a id="b3"></a>B3 — The local draft is deleted before the work is known to be safe · **critical**

`[id].vue:90` — `localStorage.removeItem(key)` runs as soon as the workflow has
been _started_. If the Vidispine import job fails, or the shape is not actually
replaced, the only copy of the user's hours of work is gone. Combined with B2
the user cannot even get back in to redo it.

**Fix:** keep the draft until the workflow is confirmed complete (submit should
return a workflow id the frontend can poll, or the handler should wait), and
keep read access to submitted items.

### <a id="b4"></a>B4 — localStorage can silently fail to save · **high**

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

### <a id="b5"></a>B5 — You can only insert a row where there is a ≥1 s gap · **high**

`canAddSegment` (`TranscriptionEditor.vue:54-60`) only shows the `+` when
`next.start >= curr.end + 1`. That is exactly R4/R5: the `+` appears "innimellom"
and nowhere else, so users cram missing sentences into the neighbouring row.
`addNewSegmentAt` (`:66`) additionally returns early when there is no previous or
next segment, so you can never insert **before the first** or **after the last**
row.

**Fix:** allow insertion between any two rows (and at both ends); derive the new
row's timing by splitting the neighbour's interval when there is no gap, and let
the user adjust start/end.

### <a id="b6"></a>B6 — One `contenteditable` per word blocks the three edits this tool exists for · **high**

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

### <a id="b8"></a>B8 — `SubmitTranscription` has no authorization check at all · **high (security)**

`backend/cmd/server/transcription.go:121-142` never calls `getEmail` or
`PermissionsForEmail`. Any request that reaches the backend can overwrite the
subtitles of any VXID. `GetTranscriptionPreview` right above it does the full
check; `GetTranscription` (`:68`) checks the tool permission but — unlike the
preview endpoint — not `inTranscriptionCollection`, so any Mediabanken-level user
can read any transcription.

**Fix:** mirror the `GetTranscriptionPreview` checks in both handlers.

### <a id="b9"></a>B9 — `clearLocalData` wipes localStorage for the whole origin · **medium**

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

| Phase                     | Scope                                                                                | Why first                                                                                             |
| ------------------------- | ------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------- |
| **0 — stop the bleeding** | [B1](#b1), [B3](#b3), [B4](#b4) (guard + strip `tokens` + debounce), [B9](#b9)       | These lose work. Nothing else matters until edits reliably survive.                                   |
| **1 — trust**             | [I1](#i1) server drafts, [I2](#i2) save/submit clarity, [B2](#b2) post-submit access | Users can see their work is safe.                                                                     |
| **2 — editing**           | [B6](#b6), [B5](#b5), [I3](#i3) 1-3, [I4](#i4) confidence                            | Delete/add a word become possible at all; undo, replace-all and confidence shading cut the work down. |
| **3 — rendering**         | [B7](#b7), [B10](#b10)                                                               | Removes the remaining "weird" behaviour.                                                              |
| **4 — polish**            | [B8](#b8) (do earlier if exposure warrants), [B11](#b11), [I4](#i4), [I5](#i5)       |                                                                                                       |

## 5. Open questions

- Is the `transcription_json` shape actually replaced on re-import, or can an
  asset end up with two `transcription_json` formats (in which case
  `GetTranscriptionJSON` picking `formats[0]` would explain R6 on its own)?
  → check a submitted asset in Cantemo.
- Should submitted transcriptions stay readable (read-only) to the volunteer who
  did them? Current behaviour removes them from `_AccessibleByTools`.
- Do we want a review step (volunteer submits → staff approves) rather than
  volunteer-submits-straight-to-Mediabanken?
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
