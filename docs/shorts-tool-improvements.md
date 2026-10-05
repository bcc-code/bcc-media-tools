# Shorts Tool — Bugs & Improvements

> Status: **§1 bugs done** (B1–B5, B7) · **[I1](#i1) transcript done** ·
> **[I3](#i3) crop guide done** · Owner: TBD
>
> Notes from a review of the shorts generation tool (`/shorts/generate`) on
> 2026-10-02. Bug fixes and the transcript panel landed 2026-10-05; [I2](#i2)
> and everything in §3 are still open.

**Scope — this is a clip-selection tool, not an editor.** The user's whole job is
to find the good 30 seconds in a 20-minute talk, mark in/out, and submit. The
cropping, subtitling and rendering all happen downstream in
`bcc-media-flows` (`workflows/export/generate_short.go`). Every improvement below
is judged on one question: **does it make finding and marking the right range
faster, or does it make the result less of a surprise?** Features that do
neither are not worth the surface area.

## 1. Bugs

### <a id="b1"></a>B1 — Nothing validates the range, and the failure is invisible · **high** · ✅ fixed

Neither `submit()` (`generate.vue:112`) nor `SubmitShort` (`shorts.go:44`)
checks that `start < end`. The workflow does validate
(`generate_short.go:59`, `InSeconds must be < OutSeconds`) but it rejects as a
Temporal `ValidationError` the user never sees — they get a success toast and are
navigated away. The submit button is also always enabled, and the <60 s rule is
only a visual warning.

**Fixed:** `isValidRange` disables both submit buttons and guards `submit()`;
`validateShortRange` in `shorts.go` rejects with `CodeInvalidArgument` and the
error toast now carries the server message. Covered by
`cmd/server/shorts_test.go`. **>60 s stays a warning** — it is editorial advice,
not something the pipeline enforces.

### <a id="b2"></a>B2 — Dragging the selection past an edge stretches it · **high** · ✅ fixed

`ShortsTimelineScrubber.vue:169` clamps `start` and `end` independently:

```ts
watch([start, end], ([s, e]) => {
  if (s < props.min) start.value = props.min;
  if (e > props.max) end.value = props.max;
});
```

A `move` drag (`:148`) adds the same delta to both. Drag the whole selection left
past 0 and `start` pins to 0 while `end` keeps travelling — a 30 s clip silently
becomes 45 s.

**Fixed:** drags now resolve against a `DragOrigin` baseline captured at
`pointerdown`, and `move` clamps the _offset_ to `[min - start, max - end]` so
the span keeps its length at both edges. The independent clamp watch remains as
a safety net for values set by the parent.

### <a id="b3"></a>B3 — Preview failure shows skeletons forever · **high** · ✅ fixed

`generate.vue:286` renders the loading state with `v-if="status != 'success'"`,
so an **error** renders a permanent shimmer. Assets with no preview shape are
common, which makes this the first thing a new user hits.

**Fixed:** the template is now a proper `success` / `error` / `loading` chain.
The error state explains the likely cause and offers **Try again** plus the
Cantemo **Generate preview** chip, the latter only for users who hold
`canCantemoPreview`.

### <a id="b4"></a>B4 — Zoom resets on every window resize · **medium** · ✅ fixed

`generate.vue:105` watches `scrubberWidth`, so any resize (or sidebar toggle)
recomputes `zoom` to fit and throws away whatever the user had set.

**Fixed:** a `zoomPinned` flag stops the refit as soon as the user moves the
slider. A "Fit" button to get back to the fitted level is still open — see
[I6](#i6).

### <a id="b5"></a>B5 — Drag uses `event.movementX` · **medium** · ✅ fixed

`ShortsTimelineScrubber.vue:146` derives the delta from `movementX`, which
accumulates rounding error and drops samples on fast drags. `seekFromEvent`
(`:125`) already does the correct thing — `clientX` relative to the track rect.

**Fixed:** all drag modes derive position from `clientX` against the track rect.
Switched to pointer events
(`pointerdown`/`pointermove`/`pointerup`/`pointercancel`) with `touch-none` on
the track, so touch and pen now work.

### <a id="b6"></a>B6 — Possible A/V drift from scene-change frame dropping · **medium, upstream, unconfirmed**

`activities/crop_shorts.go:42-47` builds
`select='not(between(t,ts,ts+0.02))'` + `setpts=N/FRAME_RATE/TB`, dropping ~0.02 s
of **video** at every detected scene change. The Norwegian audio is mapped
separately (`-map 1:a`, `:76`) with no corresponding drop. A clip with 20 scene
changes would end ~0.4 s out of sync.

May well be intentional. Worth confirming against a generated short before
treating it as a bug — it lives in `bcc-media-flows`, not here.

### <a id="b7"></a>B7 — `defineModel` narrowed the selection type to `0` · **low** · ✅ fixed

`defineModel("start", { default: 0 })` inferred the literal type `0` rather than
`number`, so every assignment to `start`/`end` failed `pnpm typecheck` — eight
errors, pre-dating this round of work. Fixed with an explicit
`defineModel<number>`; `pnpm typecheck` is clean again.

## 2. The three improvements worth doing

Discussed and prioritised 2026-10-02. Everything in §3 is secondary to these.

### <a id="i1"></a>I1 — Use the transcript to find the clip · ✅ done

Finding the quotable 30 seconds is the actual work, and scrubbing a 22-minute
video blind is the slow way to do it. The data is already there and already the
right shape: `Segments` carry `start`/`end`/`text` with nested `Words` that have
word-level `start`/`end`/`confidence` (`api.proto:188-213`).

**The blocker is permissions, not data.** `GetTranscription`
(`transcription.go:72`) requires _transcription_ permission **and** the asset
inheriting ACLs from `_AccessibleByTools` (VX-2677). A shorts user has neither.

**Needed:** a `GetShortsTranscript` RPC gated on `CanShorts()` alone — the
per-tool pattern the proto already documents at `api.proto:559-561` for
`GetShortsPreview`. ~15 lines reusing `cantemoClient.GetTranscriptionJSON`.

Degradation is free: an asset with no transcript returns an empty
`Transcription{}` rather than an error (`services/cantemo/client.go:109`).

**Built as:**

- `GetShortsTranscript` RPC (`api.proto`, handler in `shorts.go`) gated on
  `CanShorts()`. The response mapping moved to `mapTranscriptionToAPI` in
  `transcription.go` and is now shared by both handlers.
- `ShortsTranscriptPanel.vue` in a right rail beside the video; the timeline
  keeps the full width below. Stacks under the video below `lg`.
- Pure logic in `app/utils/shortsTranscript.ts`, covered by
  `test/shortsTranscript.spec.ts` (16 cases).
- **Drag across the text → sets in/out from word boundaries.** A press that
  never leaves a word is a seek instead; both keep the highlight.
- Words are indexed continuously across segments, so a selection that spans
  segments is just a pair of numbers. Segments the ASR did not time word by word
  get one entry covering the segment, so they are selectable too.
- Padding: `PRE_ROLL` 0.2 s before the first word, `TAIL` 0.15 s after the last,
  both clamped to the media. Whisper boundaries sit on the first and last sample
  of the consonant, so an unpadded cut clips audibly.
- Search highlights matching segments; Enter steps forward, shift+Enter back.
- Follow-along scrolls the active segment into view, and stands down for 3 s
  after any wheel or pointer gesture in the panel rather than trying to tell a
  programmatic scroll from a real one.
- No transcript → an empty `Transcription{}`, rendered as a neutral banner. The
  page stays fully usable.

**Permission decision — taken, not yet ratified.** Shorts users can now read the
transcript of any asset they can open in the tool. The reasoning: the transcript
is derived from the audio of a video the same permission already streams through
`GetShortsPreview`, so it exposes nothing they cannot hear by pressing play, and
it is read-only — editing stays behind the transcription permission. Reversing it
is a one-line change to the gate in `GetShortsTranscript`.

**Not built:** the word-drag gesture calls `preventDefault`, so the transcript
cannot be selected with the mouse for copying. Fine for the clip-picking job, but
worth revisiting if anyone wants to quote the text.

### <a id="i2"></a>I2 — Cut multiple shorts per video

Cutting 3–5 clips from one talk is the normal case; today that is five round
trips through the same video. The constraint is UI — this is only worth doing if
the multi-clip timeline stays as legible as the single-selection one.

Model: promote the single selection to a **list of clips**.

- Timeline renders every clip as a block; the active one has drag handles, the
  rest are outlined and dimmed. Click a block to activate it.
- Clip list below the timeline or in the rail: thumbnail, editable name, in/out,
  duration, status chip
- "Add clip" button + a key (`A`) commits the current in/out and leaves a fresh
  selection ready. The rhythm becomes scrub → I → O → A, scrub → I → O → A.
- Per-clip duration warning instead of the single global one
- Submit opens one dialog listing all clips, then shows per-clip success/failure
  in the list rather than one toast
- Overlapping clips are legitimate (variants of the same moment) — allow them,
  but hint at the overlap visually

Two things this unlocks cheaply:

- **An editable name per clip.** The workflow currently auto-names
  `<title>_short_<timestamp>` (`generate_short.go:100`), so five shorts from one
  talk are indistinguishable in Mediabanken. Needs a `Name` field threaded
  through `SubmitShortRequest` → `GenerateShortDataParams`.
- **A natural home for render status** — the clip list is already the right place
  to show rendering / done / failed, which pairs with [I3](#i3).

Backend: add `SubmitShorts` taking `repeated ShortRequest` and returning
`repeated string workflow_ids`, rather than looping client-side — partial failure
becomes one response instead of N.

Also: persist the clip list in `useLocalStorage` keyed by VX-ID. Losing one
selection to a reload is annoying; losing five cuts is a setback.

### <a id="i3"></a>I3 — Preview the actual 9:16 crop · ✅ static guide done

Users only ever see the 16:9 source, but the output is a vertical crop driven by
AI keyframe tracking. "Preview short" currently previews something that is not
what gets made.

**The real AI crop cannot be previewed pre-submit.** The keyframes come from an
external service (`SHORTS_SERVICE_URL`, `activities/shorts.go:18`) that runs on
the _already-cut_ clip, polled in a 5 s loop inside the workflow
(`generate_short.go:154-176`). Getting them early means running the same
expensive job.

**But the fallback is deterministic and free.** `buildCropFilter` with zero
keyframes (`activities/crop_shorts.go:100`) is exactly:

```
crop = floor(in_h*9/16/2)*2 wide × in_h tall, horizontally centered
```

For a 16:9 source that is the centre ~31.6 % of the width, full height — a pure
CSS overlay on the existing `<video>`. A **"Show 9:16 crop guide"** toggle that
dims outside that band answers the question people actually have ("is she going
to be in frame?") with no backend work at all.

Split accordingly:

- **Built:** a toggled centre-crop overlay (`ShortsCropGuide.vue`, geometry in
  `app/utils/shortsCrop.ts`, 7 tests). It derives the column from the source's
  real dimensions rather than assuming 16:9, since the crop is defined against
  the frame _height_, and measures against the letterboxed picture rather than
  the video element, so a 4:3 or scope source lands in the right place. A source
  already at or narrower than 9:16 is kept whole. The toggle persists in
  `localStorage`, default off.
- The button's tooltip says the guide is approximate, because it is: it matches
  the output exactly only when the shorts service returns no keyframes. Worth
  keeping that wording honest if the overlay is ever made more prominent.
- **Later:** seeing the real tracked crop becomes a _post-generation review_
  step. The keyframes are already returned in `GenerateShortResult`
  (`generate_short.go:222`), so a review view could replay the tracked crop over
  the source.

## 3. Smaller improvements

### I4 — Submit is fire-and-forget

`SubmitShort` returns `Void` and discards the Temporal handle. The user gets a
toast and then has no idea whether the short succeeded, failed, or is still
rendering ten minutes later. `StartExport` already returns `workflow_ids`
(`api.proto:301-303`) — do the same, then add a "My shorts" list on `/shorts`
polling status. Prerequisite for the status chips in [I2](#i2).

### I5 — Orientation and sharing

- **No asset title anywhere on the page.** You see a video and a VX-ID in the
  URL. `ResolveAssets` already returns titles.
- **In/out are not in the URL.** A reload loses the work and you cannot send a
  colleague "this exact range".
- **The confirm dialog says nothing about what is being submitted** — show
  title, range and duration.
- **`/shorts` only accepts a VX-ID.** Accept a pasted Mediabanken URL and show
  recent assets.

### I6 — Keyboard and timeline controls

- Arrow keys step 1 s and nothing else. Add shift+arrow for 10 s, `,`/`.` for
  frame steps, and `J/K/L` — standard for anyone coming from a real NLE.
- `useVideoKeyboardControls` has no input-field guard. Harmless today (no inputs
  on the page) but [I2](#i2)'s clip names and [I1](#i1)'s search box both break
  it.
- Zoom slider is unlabelled and sits far from the timeline. Ctrl/Cmd+scroll to
  zoom at the cursor plus a "Fit" button is what people reach for.

### I7 — Show scene cuts on the timeline

The workflow already runs `FFmpegGetSceneChanges` (`generate_short.go:125`) —
but only _after_ submission. Exposing those as timeline markers with snap-to-cut
for in/out would stop people cutting mid-shot, using data we already compute.
Needs an endpoint that can produce them pre-submit.

### I8 — Audio and subtitles are hardcoded

The workflow takes Norwegian audio only (`generate_short.go:189-192`) and subtitle
burn-in is commented out (`:200`). A source without a `nor` audio file
silently produces a short with no audio. At minimum warn when `nor` is missing;
better, offer the choice the way the export tool does.

### I9 — i18n gaps

- `useTools.ts:46-52` hardcodes English `"Shorts generation"` /
  `"Generate shorts from existing videos"` while every other tool uses `t()`.
  There are no `tools.shorts` keys in either locale.
- `shorts.generation.description` in `nb.json`: "En enkelt verktøy" → **"Et**
  enkelt verktøy".

## 4. Suggested order of work

| Phase                 | Scope                                       | Why                                                                   |
| --------------------- | ------------------------------------------- | --------------------------------------------------------------------- |
| **0 — bugs** ✅       | [B1](#b1)–[B5](#b5), [B7](#b7)              | Small, self-contained, and B1/B3 are the ones users actually hit.     |
| **1 — transcript** ✅ | [I1](#i1)                                   | Biggest change to how the tool is used; the permission work is small. |
| **2 — crop guide** ✅ | [I3](#i3) (static overlay only)             | An afternoon, and it stops the output being a surprise.               |
| **3 — status**        | [I4](#i4)                                   | Prerequisite for the clip-list status chips.                          |
| **4 — multi-cut**     | [I2](#i2)                                   | Most UI surface; benefits from I4 having landed.                      |
| **5 — polish**        | [I5](#i5)–[I9](#i9), [B6](#b6) if confirmed |                                                                       |

## 5. Open questions

- Should >60 s **block** submission or stay a warning? ([B1](#b1))
- Is widening transcript access to shorts users acceptable? ([I1](#i1))
- Is the scene-change frame dropping in `CropShortActivity` intentional?
  ([B6](#b6)) → check a generated short for drift.
- Should `ModelSize` / `DebugMode` (hardcoded `"n"` / `false` in `shorts.go:66`)
  be exposed as advanced options, or stay fixed?
- Does subtitle burn-in want to come back? It is commented out upstream, and
  shorts normally need burned subs. ([I8](#i8))

## 6. Files involved

```
frontend/app/pages/shorts/generate.vue                        player, in/out state, submit
frontend/app/pages/shorts/index.vue                           VX-ID entry point
frontend/app/components/shorts/ShortsTimelineScrubber.vue     filmstrip, selection, ruler, playhead
frontend/app/composables/useVideoKeyboardControls.ts          space / arrows / I / O
frontend/app/composables/useTools.ts                          tool list entry (hardcoded strings)
frontend/app/pages/vault/[id].vue                             "Create short" entry point
backend/cmd/server/shorts.go                                  GetShortsPreview, SubmitShort
backend/cmd/server/vault_proxy.go                             /vault/thumbnail (filmstrip frames)
api/v1/api.proto                                              SubmitShortRequest, Preview, ShortsPermission
```

Upstream in `bcc-media-flows`:

```
workflows/export/generate_short.go                            orchestration, naming, audio/subtitle choices
activities/crop_shorts.go                                     crop filter, scene-change select, ffmpeg args
activities/shorts.go                                          SHORTS_SERVICE_URL job submit/poll, Keyframe
services/cantemo/client.go                                    GetTranscriptionJSON, GetPreviewUrl
```
