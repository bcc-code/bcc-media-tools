# Shorts Tool — Bugs & Improvements

> **Done:** §1 bugs (B1–B5, B7, B8) · [I1](#i1) transcript · [I3](#i3) crop
> guide · [I2](#i2) multi-cut
>
> **Next:** [I4](#i4) status polling — the backend half already landed with
> [I2](#i2); what is left is a "my shorts" view that polls the returned ids
>
> **Owner:** TBD
>
> Notes from a review of the shorts generation tool (`/shorts/generate`) on
> 2026-10-02. Everything marked done landed on 2026-10-05 and is committed
> (`795bf5f`, `3b3aa9e`, plus the crop guide). Nothing done here has been used
> in a real editing session yet — see [Still to confirm](#confirm).
>
> [B6](#b6) is the one open item that affects **already-generated** shorts
> rather than future work, and it costs ten minutes to settle.

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

### <a id="b8"></a>B8 — Shortcuts fired while typing in the transcript search · **high** · ✅ fixed

`useVideoKeyboardControls` bound bare letters and space with no check on where
the keystroke landed, so a space in the transcript search box played the video
and an "o" set the out point. Introduced by [I1](#i1)'s search field and found
while adding the `A` shortcut for [I2](#i2).

**Fixed:** every shortcut now ignores events whose target is an input, textarea,
select or contenteditable.

## 2. The three improvements worth doing

Discussed and prioritised 2026-10-02. All three are done; everything in §3 was
always secondary to them.

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

### <a id="i2"></a>I2 — Cut multiple shorts per video · ✅ done

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

**Built as:**

- `useShortsClips` holds the list, the active clip, and localStorage per asset.
  **There is no separate draft range**: the active clip _is_ what the timeline
  and transcript edit, which removes the failure where someone trims a range,
  submits, and their last clip is missing because they never pressed Add.
- The page keeps working through `activeStart` / `activeEnd` writable computeds,
  so the timeline, transcript and crop guide did not have to learn about lists.
- `ShortsClipList.vue` is a bordered panel in the right rail under the
  transcript: a vertical row per clip with its number, label, in–out, length, a
  warning tint past a minute, an overlap hint, and remove. It started as a strip
  of chips above the timeline and moved after a round of feedback — see the
  [UI pass](#ui). `ShortsTimelineScrubber` draws every clip — inactive ones as
  bands above the dimming, the active one keeping its handles — and clicking a
  band activates it.
- **Clips are listed in creation order, not time order.** Sorting by start time
  renumbers the list whenever a clip is trimmed earlier, so the row the user was
  looking at changes number under their hands. The number says which clip it is;
  the timecode says where it sits.
- `A` adds a 30 s clip at the playhead. On first load the list seeds with one
  clip spanning the whole asset, which is how the editor behaved before.
- `SubmitShorts` RPC takes `repeated ShortClip` and returns a per-clip
  `{workflow_id, error}`, so one bad clip reports itself and the rest still run.
  `SubmitShort` now delegates to the same `startShort`, so the workflow
  parameters live in one place.
- Pure logic in `app/utils/shortsClips.ts`, 25 tests. `sanitiseClips` hardens the
  localStorage read: clips come back from storage in whatever shape an older
  build or a hand edit left, and a bad entry must not take the editor down.

**Labels come from the transcript, not from the user.** Each row shows the
opening words spoken inside the clip (`labelForRange`), which identifies it
better than anything someone would be willing to type and costs them nothing. It
reads from where the _range_ starts rather than the segment, so a clip beginning
mid-sentence still labels from the right word, and falls back to a muted "no
transcript" line.

**Still blocked upstream:** the label is local to the editor. The _output_ name
stays `<title>_short_<timestamp>` because `GenerateShortDataParams`
(`generate_short.go:28`) has no `Name` field, so five shorts from one talk are
still indistinguishable in Mediabanken. Needs a `bcc-media-flows` change first;
the label is already there to send once there is somewhere to send it.

**Still to do:** the clip list is the right home for render status, which is
[I4](#i4) — the returned workflow ids are already there to poll.

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

## <a id="ui"></a>2b. UI pass after going full-width (2026-10-05)

The page was widened to fill the window, which broke several things that had
been fine inside `max-w-7xl` and exposed assumptions that did not survive a
96-minute asset.

**Layout is a viewport-height shell.** `lg:h-[calc(100dvh-var(--header-height))]`
with the media row as the only flexing child, so the timeline can never be pushed
below the fold however wide the window gets. The `--header-height` variable is
published by `layouts/default.vue` precisely for this and is already used by the
transcription pages — `h-dvh` alone is one header too tall. Below `lg` the page
scrolls normally, since a stacked layout has no height to redistribute.

**The clip actions moved under the video.** The readout and the buttons that
change it were at opposite ends of a 2000 px row, with the buttons sitting under
the transcript they have nothing to do with. They now live in the video column
directly under the player, with the 9:16 guide separated off as a view toggle
rather than a clip action.

**Activating a clip seeks to it.** `activate` only set `activeId`, so picking a
clip at 27:52 while the playhead sat at 80:00 left both the player and the
timeline where they were and the chosen clip off-screen.

**Inactive clip bands have a floor width** (`MIN_BAND_PX`). A 35 s clip on a
96-minute asset is ~12 px zoomed out and sub-pixel beyond that, so clips were
invisible on the timeline. The _active_ band keeps its true width deliberately:
widening it would put its edges out of step with the dimming, which is what shows
where the clip really starts and ends.

**Timestamps are hour-aware.** `formatClock` takes the asset length and uses
`h:mm:ss` for every value from an asset over an hour, so the transcript, the clip
rows and the timeline ruler agree. Lengths pass no total and stay `m:ss`, so a
35 s clip reads `0:35` and not `0:00:35`.

> A correction worth keeping: the original `formatClock` deliberately did _not_
> wrap minutes into hours, on the reasoning that assets over an hour would be
> rare. They are not — ordinary conference recordings here run past 90 minutes.
> The result was the transcript reading `80:15` while the readout for the same
> moment read `01:20:19.110`. Assumptions about typical asset length should be
> checked against VAULT rather than guessed.

## 3. Smaller improvements

### <a id="i4"></a>I4 — Submit is fire-and-forget

`SubmitShort` returns `Void` and discards the Temporal handle. The user gets a
toast and then has no idea whether the short succeeded, failed, or is still
rendering ten minutes later. `StartExport` already returns `workflow_ids`
(`api.proto:301-303`) — do the same, then add a "My shorts" list on `/shorts`
polling status. Prerequisite for the status chips in [I2](#i2).

### <a id="i5"></a>I5 — Orientation and sharing

- **No asset title anywhere on the page.** You see a video and a VX-ID in the
  URL. `ResolveAssets` already returns titles.
- **In/out are not in the URL.** A reload loses the work and you cannot send a
  colleague "this exact range".
- **The confirm dialog says nothing about what is being submitted** — show
  title, range and duration.
- **`/shorts` only accepts a VX-ID.** Accept a pasted Mediabanken URL and show
  recent assets.

### <a id="i6"></a>I6 — Keyboard and timeline controls

- Arrow keys step 1 s and nothing else. Add shift+arrow for 10 s, `,`/`.` for
  frame steps, and `J/K/L` — standard for anyone coming from a real NLE.
- `useVideoKeyboardControls` has no input-field guard. Harmless today (no inputs
  on the page) but [I2](#i2)'s clip names and [I1](#i1)'s search box both break
  it.
- Zoom slider is unlabelled and sits far from the timeline. Ctrl/Cmd+scroll to
  zoom at the cursor plus a "Fit" button is what people reach for.

### <a id="i7"></a>I7 — Show scene cuts on the timeline

The workflow already runs `FFmpegGetSceneChanges` (`generate_short.go:125`) —
but only _after_ submission. Exposing those as timeline markers with snap-to-cut
for in/out would stop people cutting mid-shot, using data we already compute.
Needs an endpoint that can produce them pre-submit.

### <a id="i8"></a>I8 — Audio and subtitles are hardcoded

The workflow takes Norwegian audio only (`generate_short.go:189-192`) and subtitle
burn-in is commented out (`:200`). A source without a `nor` audio file
silently produces a short with no audio. At minimum warn when `nor` is missing;
better, offer the choice the way the export tool does.

### <a id="i9"></a>I9 — i18n gaps

- `useTools.ts:46-52` hardcodes English `"Shorts generation"` /
  `"Generate shorts from existing videos"` while every other tool uses `t()`.
  There are no `tools.shorts` keys in either locale.
- `shorts.generation.description` in `nb.json`: "En enkelt verktøy" → **"Et**
  enkelt verktøy".

## 4. Suggested order of work

| Phase                 | Scope                                       | Why                                                                   |
| --------------------- | ------------------------------------------- | --------------------------------------------------------------------- |
| **0 — bugs** ✅       | [B1](#b1)–[B5](#b5), [B7](#b7), [B8](#b8)   | Small, self-contained, and B1/B3 are the ones users actually hit.     |
| **1 — transcript** ✅ | [I1](#i1)                                   | Biggest change to how the tool is used; the permission work is small. |
| **2 — crop guide** ✅ | [I3](#i3) (static overlay only)             | An afternoon, and it stops the output being a surprise.               |
| **3 — status**        | [I4](#i4)                                   | Prerequisite for the clip-list status chips.                          |
| **4 — multi-cut** ✅  | [I2](#i2)                                   | Most UI surface; benefits from I4 having landed.                      |
| **5 — polish**        | [I5](#i5)–[I9](#i9), [B6](#b6) if confirmed |                                                                       |

## <a id="confirm"></a>5. Still to confirm in use

Everything in §2 was verified by tests, typecheck and build, but **none of it
has been through a real editing session**. These are the parts most likely to
need adjusting once someone actually cuts a short with them:

| Thing                   | Why it may be wrong                                                                                                                                                    |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Follow-along pause      | Playback scrolling stands down for 3 s after a wheel or pointer gesture. The number is a guess; nobody has watched it run during a real read-through.                  |
| Scroll-to-selection     | Scrolls the moved edge to the top (`block: "start"`, after feedback). For a clip already on screen it may still jump when it should sit still — `"nearest"` would not. |
| Pre-roll / tail         | 0.2 s and 0.15 s around a word selection. Tuned by reasoning about where Whisper puts boundaries, not by listening to the result.                                      |
| Highlight strength      | `30%` light / `50%` dark, after one round of feedback. Fine on the two assets seen so far.                                                                             |
| Crop guide accuracy     | Exact only when the shorts service returns no keyframes. Nobody has compared the guide against a generated short's real framing.                                       |
| Long transcripts        | No virtualisation. ~450 segments for a 38-minute talk is fine; a two-hour recording is ~11k word spans and may scroll badly. Depends on typical length.                |
| Copying transcript text | The word-drag gesture calls `preventDefault`, so the text cannot be selected for copying. Deliberate, but untested against how people actually work.                   |

## 6. Open questions

- ~~Should >60 s **block** submission or stay a warning?~~ **Answered
  (2026-10-05): stays a warning.** It is editorial advice, not a constraint the
  pipeline enforces, so [B1](#b1) blocks only on `in >= out`.
- **Taken, not ratified:** transcript access was widened to shorts users
  ([I1](#i1)), on the grounds that the transcript is derived from audio the same
  permission already streams. Still worth a conscious yes/no from whoever owns
  the permission model — reversing it is a one-line change to the gate in
  `GetShortsTranscript`.
- **Open, and worth 10 minutes:** is the scene-change frame dropping in
  `CropShortActivity` intentional? ([B6](#b6)) It drops ~0.02 s of video per
  scene change without dropping audio, so if it is a bug every short generated
  so far drifts. → play a generated short with many cuts to the end and listen
  for sync.
- Should `ModelSize` / `DebugMode` (hardcoded `"n"` / `false` in `shorts.go:66`)
  be exposed as advanced options, or stay fixed?
- Does subtitle burn-in want to come back? It is commented out upstream, and
  shorts normally need burned subs. ([I8](#i8))

## 7. Files involved

```
frontend/app/pages/shorts/generate.vue                        player, in/out state, submit
frontend/app/pages/shorts/index.vue                           VX-ID entry point
frontend/app/components/shorts/ShortsTimelineScrubber.vue     filmstrip, selection, ruler, playhead
frontend/app/components/shorts/ShortsTranscriptPanel.vue      transcript rail, word drag, search, follow
frontend/app/components/shorts/ShortsCropGuide.vue            9:16 overlay
frontend/app/components/shorts/ShortsClipList.vue             clip rail panel
frontend/app/composables/useShortsClips.ts                    clip list state, active clip, localStorage
frontend/app/utils/shortsClips.ts                             clip ranges, overlap, sanitising, formatClock
frontend/test/shortsClips.spec.ts                             28 cases
frontend/test/locales.spec.ts                                 en/nb key parity, plural forms
frontend/app/utils/shortsTranscript.ts                        flatten, selection -> range, highlight, clip labels
frontend/app/utils/shortsCrop.ts                              crop geometry
frontend/test/shortsTranscript.spec.ts                        36 cases
frontend/test/shortsCrop.spec.ts                              7 cases
frontend/app/composables/useVideoKeyboardControls.ts          space / arrows / I / O
frontend/app/composables/useTools.ts                          tool list entry (hardcoded strings)
frontend/app/pages/vault/[id].vue                             "Create short" entry point
backend/cmd/server/shorts.go                                  GetShortsPreview, SubmitShort, GetShortsTranscript
backend/cmd/server/shorts_test.go                             range validation
backend/cmd/server/transcription.go                           mapTranscriptionToAPI (shared with shorts)
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
