# Design System Migration Plan

Migrating `bcc-media-tools/frontend` off **Nuxt UI** onto the custom design system
used in **`bcc-media-platform/admin-web`**. This is something we want to do so that our internal tools are more unified.

## Current state (`main`, 2026-06-22)

- Nuxt UI (`@nuxt/ui`) is an active module and fully in use: **~31 `.vue` files, ~190 `<U*>` instances**.
- `app/assets/css/main.css` is 23 lines — imports tailwind + `@nuxt/ui`, sets the
  Archivo font and `--ui-primary` (`#004e48` light / `#a0cec8` dark). No design tokens.
- No `app/components/design/` directory, no `Design*` components.
- A prior port attempt was reverted; the migration has effectively not started beyond brand color + font.

### `<U*>` usage inventory (migration leverage, highest first)

| Component    | Count | Component   | Count |
| ------------ | ----- | ----------- | ----- |
| `UButton`    | 47    | `UIcon`     | 10    |
| `USkeleton`  | 26    | `UModal`    | 7     |
| `USwitch`    | 18    | `UInput`    | 5     |
| `USelect`    | 16    | `UCard`     | 3     |
| `UFormField` | 16    | `UAlert`    | 3     |
| `UCheckbox`  | 11    | (long tail) | …     |

## Source system (`admin-web`)

- **Stack:** Tailwind 4 + **Ark UI** (`@ark-ui/vue` ^5) + **`cva`** (`1.0.0-beta.4`). **No Nuxt UI.**
- **Tokens:** `app/assets/css/main.css` (~293 lines). A `@theme` block maps `--color-*`,
  typography (`text-heading-*`, `text-title-*`, `text-body-*`, `text-caption-*`) and shadows
  onto runtime `--ds-*` variables, themed for light and `.dark`.
- **26 `Design*` components:** Button, Input, Textarea, Select, Switch, Badge, Checkbox-via-Switch,
  Dialog, Tooltip, ToastProvider, Avatar, Banner, EmptyState, ErrorState, LoadingState, ViewState,
  ProgressCircle, DatePicker, RadioGroup, SegmentGroup, StatusIndicator, TagsInput, Table, Tour,
  AgeRating(+Picker), ConfirmProvider.
- **Composables:** `useBrand`, `useToast`, `useConfirm`, `useProductTour`.
- Components are CVA-based (`variant` / `intent` / `size` props compiled to token classes).
- Source paths:
    - Tokens: `bcc-media-platform/admin-web/app/assets/css/main.css`
    - Components: `bcc-media-platform/admin-web/app/components/design/Design*.vue`

## Guiding principle

**Decouple "adopt the visual language" from "remove Nuxt UI."** Do tokens first so every
existing `<U*>` inherits the look for free; swap components page-by-page; delete Nuxt UI last.
A previous big-bang page sweep was reverted — validate component-by-component, never all at once.

---

## Stage 1 — Token layer (do first; low risk, reversible)

1. Port admin-web's `@theme` block into `app/assets/css/main.css` (or a new `theme.css` imported
   by it): `--color-*` / `--ds-*` light + `.dark` vars, the typography scale, and shadows.
2. Add a shim that **rebinds Nuxt UI's `--ui-*` variables onto the new tokens**
   (`--ui-primary`, `--ui-bg`, `--ui-text`, radii, etc.).
3. Result: all ~190 `<U*>` components inherit the admin-web look with **zero component edits**.
   This delivers the bulk of the visual parity and is trivially revertible.

## Stage 2 — Component infrastructure

1. Add `@ark-ui/vue` and `cva` to `package.json`.
2. Create `app/components/design/`. Port `Design*` components **on demand**, keeping their
   CVA structure intact.
3. Port composables as needed. **Rename to avoid clashing** with Nuxt UI while both coexist —
   e.g. Nuxt UI's `useToast`/`useModal` are still live, so expose the ported toaster as
   `useDesignToaster` (or similar).

## Stage 3 — Incremental sweep

Replace `U*` → `Design*` **one small page at a time**, eyeball-validating each before continuing.
Order by leverage: `UButton` → `UInput`/`UFormField` → `USelect`/`USwitch`/`UCheckbox` → `UModal`.
`USkeleton`/`UIcon` are cheap. Watch for known regression traps (e.g. multi-`select` rendering
"N selected" instead of chips).

## Stage 4 — Remove Nuxt UI

Only once `grep -r '<U[A-Z]' app/` returns nothing:

1. Remove `@nuxt/ui` from `nuxt.config.ts`.
2. Remove the `@import "@nuxt/ui"` and the `--ui-*` rebinding shim from `main.css`.
3. Drop the `@nuxt/ui` dependency from `package.json`.

---

## Pilot (this iteration): one page end-to-end

**Pilot surface: `app/pages/export.vue` + `app/components/export/ExportForm.vue`.** This is a
rich, representative slice rather than a minimal one — converting it surfaces most of the patterns
the rest of the sweep will need. Combined `<U*>` usage:

| `export.vue`   | `ExportForm.vue`           |
| -------------- | -------------------------- |
| `UCard` ×1     | `UButton` ×9               |
| `UIcon` ×2     | `UCheckbox` ×8             |
| `USkeleton` ×6 | `USelect` ×2               |
|                | `UFormField` ×2            |
|                | `UModal` ×1                |
|                | `UTextarea` ×1             |
|                | `UCard` ×1, `USkeleton` ×1 |

It also exercises `useToast` (in `export.vue`), so the toast-composable rename gets validated here.

### Components needed for the pilot

- **Direct `Design*` ports:** `DesignButton`, `DesignCheckbox` (admin-web models this via Switch —
  confirm the checkbox variant), `DesignSelect`, `DesignTextarea`, `DesignDialog` (for `UModal`).
- **Gaps to resolve in the pilot** (no direct admin-web equivalent):
    - **`FormField`** — admin-web folds label/error into the input components; decide the
      label + validation-error pattern.
    - **`UCard`** — no `Design*` equivalent; use plain markup + surface/shadow tokens.
    - **`USkeleton`** — no `Design*` equivalent; either keep `USkeleton` temporarily (Stage 1 tokens
      already restyle it) or build a token-based skeleton.
    - **`UIcon`** — replace with `<Icon>` (`@nuxt/icon`, `tabler:*`).
- **Composable:** rename `useToast` → `useDesignToaster` and wire `DesignToastProvider`.

### Pilot steps

1. **Stage 1 tokens** — port tokens + Nuxt UI var shim; verify the whole app still renders and
   visibly shifts toward the admin-web look.
2. **Port the needed components** (list above), deciding the `FormField`/`Card`/`Skeleton` patterns.
3. **Convert `export.vue` + `ExportForm.vue`** fully to `Design*` / tokenized markup.
4. **Validate** — run the app, exercise the export form (incl. the modal + toast paths), compare
   side-by-side against admin-web, confirm no functional or visual regressions.
5. **Decide** — if the look + ergonomics are right, proceed to Stage 3 sweep across remaining pages
   in leverage order. If not, adjust the ported components before spreading them.

Alternative (lighter) pilots, if the above proves too broad: `app/pages/admin.vue`
(`UButton` ×3 + `UInput`) or `app/pages/shorts/index.vue` (`UButton`/`UInput`/`UFormField`/`UForm`).

## Risks / watch-items

- **No Nuxt UI `FormField` analog** — settle the label/error pattern in the pilot.
- **`useToast` name clash** — rename the ported composable; Nuxt UI's is still used elsewhere.
- **Multi-select chips** — `DesignSelect` `multiple` mode previously regressed to "N selected".
- **Icons** — standardize on Tabler (`tabler:*` via `@nuxt/icon`); don't reintroduce
  `heroicons:*` / `i-lucide-*`. `svg-spinners:*` (animated) is fine.
- **No big-bang** — a full-page sweep was reverted once; keep changes page-scoped and validated.

---

## Progress log

### 2026-06-22 — Stage 1 + pilot (export page) — DONE (built + typechecks; visual review pending)

**Dependencies added** (`frontend/package.json`): `@ark-ui/vue@^5` (5.37.2), `cva@1.0.0-beta.4`.

**Stage 1 — token layer (`app/assets/css/main.css`, rewritten):**

- Ported admin-web's full `@theme` token block: `--color-*` → `--ds-*` indirection, typography
  scale (`text-heading/title/body/caption-*`), `--ease-out-expo`, `--shadow-resting/floating`.
- **Only the BCC brand is baked in.** Dropped admin-web's `data-brand` switching + `useBrand`
  entirely. BCC primary is hard-set in `:root` (`#004e48`) and `.dark` (`#a0cec8`); the rest of the
  light/dark `--ds-*` values come straight from admin-web.
- **Nuxt UI bridge:** a `:root, .dark { --ui-*: var(--color-*) }` block rebinds Nuxt UI's variables
  onto the tokens, so all remaining `<U*>` components + Nuxt UI utilities (`text-muted`, `bg-default`,
  `border-default`, …) inherit the BCC look with no edits. Placed after `@import "@nuxt/ui"` so it
  wins at equal specificity in both themes.
- Kept the existing `html/body/#__nuxt` flex layout rules. Did **not** add `@custom-variant dark`
  (Nuxt UI already provides the dark variant; design components flip via `--ds-*`, not `dark:`).

**Stage 2 — components ported** (`app/components/design/`, all auto-imported as `<DesignX>` because
Nuxt dedupes the `design/` dir prefix):

- `DesignButton.vue` — admin-web port. Extensions: added `type` prop (button/submit/reset) and a
  `loading` spinner (`svg-spinners:ring-resize`); admin-web only disabled on loading.
- `DesignSelect.vue` — adapted from admin-web: single-value `string` v-model (not `string[]`) and
  accepts `(string | {label,value})[]` items, to match Nuxt UI `USelect` ergonomics. Trigger is
  `w-full`. Teleports to `#teleports` (Nuxt provides this target automatically).
- `DesignTextarea.vue` — straight admin-web port (Ark `Field`).
- `DesignDialog.vue` — straight admin-web port (Ark `Dialog`). Has `title`/`description` props +
  default slot; **no `#footer` slot** (admin-web doesn't) — put footer buttons in the default slot.
- `DesignCheckbox.vue` — **NEW, not in admin-web.** Built on Ark UI `Checkbox`, token-styled.
  Supports `v-model` (boolean), `label` prop, `#label` slot, `disabled`, `ariaLabel`. Decision:
  the export form is checkbox-heavy and checkboxes (not switches) are semantically right, so a
  checkbox was worth adding rather than forcing `DesignSwitch`.
- `DesignToastProvider.vue` + `composables/useDesignToaster.ts` — Ark toaster. **Renamed**
  `useToast` → `useDesignToaster` to avoid clashing with Nuxt UI's `useToast`. The composable
  returns the unwrapped instance (`useState(...).value`) so callers use `toaster.create(...)`.
  Provider mounted once in `app/app.vue` (inside `<UApp>`).

**Stage 3 (pilot) — converted files:**

- `app/pages/export.vue` — `useToast`→`useDesignToaster` (`toast.add({color})` → `toaster.create({type})`,
  dropped explicit `icon` — the provider derives it from `type`); `UIcon`→`Icon`; `UCard` trigger
  links → token markup (`gradient-border bg-surface-raise shadow-resting`). Kept `USkeleton`.
- `app/components/export/ExportForm.vue` — full template swap: `UTextarea`→`DesignTextarea`,
  `UButton`→`DesignButton`, `UCheckbox`→`DesignCheckbox`, `USelect`→`DesignSelect`, `UModal`→
  `DesignDialog`, `UFormField`→`<label>` + control, `UCard`→token div. Kept `USkeleton`.
  Nuxt UI semantic utilities converted to tokens (`text-highlighted`→`text-text-default`,
  `text-muted`→`text-text-muted`, `text-error`→`text-semantic-error`,
  `text-warning`→`text-semantic-warning`, `bg-default`→`bg-surface-default`,
  `border-default`→`border-border-1`, `divide-default`→`divide-border-1`).

**Button variant mapping used** (Nuxt UI → Design): `ghost`→`tertiary`, `outline`/`subtle`→
`secondary`, default primary→`primary`; sizes `xs`/`sm`→`small`, `lg`→`large`.

**Correction (post-review):** the bridge originally also set `--ui-radius: 0.5rem`, which doubled
the corner radius on every Nuxt UI component (default is `0.25rem`). Removed — don't set
`--ui-radius`; the token system has no radius variable and `Design*` components use explicit
`rounded-*` classes. Leave Nuxt UI's radius default alone.

**Verification:** `pnpm build` ✅, `pnpm typecheck` ✅ (exit 0). Runtime smoke test: `/export`
returned 200 with no SSR errors in the dev log. **Visual / interaction review by a human is still
pending** (step 4) — especially: checkbox look, select dropdown, dialog, toast appearance,
sticky-bar primary button, and dark mode.

### 2026-06-22 — focus ring + checkbox tweak

- **Focus ring convention added.** All `Design*` components get a 3px keyboard focus ring via the
  `--color-focus-ring` token (black light / white dark). Implemented as one shared rule in
  `main.css`: `.ds-focus-ring:focus-visible, .ds-focus-ring[data-focus-visible]` → `outline: 3px
solid var(--color-focus-ring); outline-offset: 2px`. Covers native `:focus-visible` and Ark's
  `[data-focus-visible]` attribute. Used `outline` (not box-shadow) to avoid conflicting with
  `shadow-resting`/`gradient-border`. **New components: add the `ds-focus-ring` class to the
  focusable element.** Applied to Button, Checkbox (control), Select (trigger), Textarea, Dialog
  (close), ToastProvider (close).
- `DesignCheckbox` indicator centering + checkbox alignment fixed by the user.

### 2026-06-23 — vb-export page migrated

`app/pages/vb-export.vue` + `app/components/vb-export/VbExportForm.vue` converted, mirroring the
export page (it's structurally a subset — same component set, no quick-action buttons):
`useToast`→`useDesignToaster`, `UIcon`→`Icon`, `UTextarea`→`DesignTextarea`, `UButton`→`DesignButton`,
`UCheckbox`→`DesignCheckbox`, `USelect`→`DesignSelect`, `UFormField`→`<label>`+control,
`UModal`→`DesignDialog`; Nuxt UI semantic utilities → tokens. `USkeleton` kept. No new components
needed (all reused from the export pilot). `pnpm typecheck` ✅; only `<USkeleton>` remains in these
files. Pending human visual review.

**Pilot-era select/button tweaks that landed (apply to future pages too):**

- `DesignSelect`: dropdown `min-w-[var(--reference-width)]` (≥ trigger width); removed the trigger
  hover state entirely (BCC dark `surface-indent` is translucent black → looked wrong).
- Quick-action "secondary" buttons that sit on a plain surface can disappear in dark mode (secondary
  fill is `surface-indent`); fix per-button with `class="border border-border-1"` rather than
  changing the `DesignButton` variant.
- Focus ring: every interactive `Design*` element carries `ds-focus-ring` (see prior entry).

### 2026-06-23 — confirm single-asset exports too (behavior change, not migration)

Investigated a report of "export started without a confirmation dialog." Root cause: the
export-trigger logic (`attemptExport`) was **unchanged by the migration** (git diff of the script
is empty) — confirmation only ever gated _bulk_ mode; single-asset exports have always fired
immediately. Verified `DesignDialog` opens correctly via `v-model:open` in isolation, so not a
dialog regression.

Per user decision, single-asset exports now also confirm. Both `ExportForm.vue` and
`VbExportForm.vue`: `attemptExport()` now always opens the dialog; added `confirmTitle` +
branched `confirmMessage` computeds (bulk vs single); dialog confirm-button label is bulk/single
aware. New i18n keys `export.confirmTitle`/`confirmMessage` and `vbExport.confirmTitle`/
`confirmMessage` in `en.json` + `nb.json`. `pnpm typecheck` ✅.

### 2026-06-23 — /admin page migrated (the previously-reverted page)

Migrated `app/pages/admin.vue` + `app/components/admin/{AdminPermissionView,AdminPermissionFilter,AdminPermissionViewSection}.vue`. No `U*` left in the admin tree; `pnpm build` + `typecheck` ✅.

**New components ported (from admin-web):**

- `DesignInput.vue` — Ark `Field`. Extensions: `leadingIcon` prop + `#trailing` slot (the filter
  search box uses both); `ds-focus-ring`. Single-line input; supports text/email/url/date/time/
  search/password.
- `DesignSwitch.vue` — Ark `Switch`. Extension: `description` prop (label + muted description
  stacked; switch top-aligned via `items-start`/`mt-0.5`); `ds-focus-ring`.

**`DesignSelect.vue` extended for multi-select** (was single-value only):

- New `multiple` prop. Model is `string | string[]` — single binds a string, multi binds a
  `string[]`. Ark always works arrays internally; bridged via an `arrayModel` computed.
- **Display decision (user-chosen): comma-separated labels, truncated with ellipsis** on overflow
  (single line, fixed height). NOT chips, NOT "N selected" (the prior reverted attempt used the
  count — that's what looked wrong). Placeholder shows in `text-text-hint` when empty.
- Width: pass width via `class` on `<DesignSelect>` — it falls through to Ark `Select.Root` (a real
  `<div>`); the trigger is `w-full` and fills it. Used `w-32`/`w-24` (filter) and `w-full
max-w-prose` (permissions).

**Mapping notes for admin:** `UButton variant="ghost"`→`tertiary`, `color="error"`→`intent="danger"`,
`variant="soft"`→`secondary`, `block`→`class="w-full"`; `USwitch`→`DesignSwitch` (1:1, all
self-closing); `UFormField`→`<label>`+control; utilities → tokens (`bg-default`→`bg-surface-default`,
`bg-muted`→`bg-surface-indent`, `border-accented`→`border-border-1`, `divide-default`→
`divide-border-1`). Kept the `motion-v` animations and grid layout untouched.

**Needs human review (esp. the bits that caused the last revert):** the 6 multi-selects
(comma display + truncation in the narrow `w-24`/`w-32` filter selects), the 15 switches with
descriptions, the filter search input (leading icon + trailing Clear button), and dark mode.

### 2026-06-23 — shorts/index migrated (UForm → native form + zod)

`app/pages/shorts/index.vue`: replaced `UForm` (Nuxt UI form + schema) with a native
`<form @submit.prevent>` that runs the same zod schema via `safeParse` on submit; errors surface
through `DesignInput`'s `:invalid` + `:error-text` (no `UFormField`). `UButton block` → `class="w-full"`.
No new components. Removed the `@nuxt/ui` `FormSubmitEvent` import. `pnpm typecheck` ✅.

**Pattern for `UForm` elsewhere:** there's no Design form component — use a native `<form>`, validate
with the existing schema (`safeParse`) on submit, and feed the first issue message into the field's
`error-text`. Reuse this for other `UForm` pages.

### 2026-06-23 — shorts/generate migrated (+ DesignSlider)

`app/pages/shorts/generate.vue` (the biggest page, 18 `U*`): `useToast`→`useDesignToaster`;
6 `UButton`→`DesignButton` (`soft`→`secondary`, header submit uses `icon="tabler:send"` so the
inline `UIcon` is dropped); 2 `UModal`→`DesignDialog` (the manual popup used `#body` → now plain
default-slot `<img>`); `USlider`→new `DesignSlider`; utilities→tokens (`bg-default`→
`bg-surface-default`, `text-muted`/`text-dimmed`→tokens). `USkeleton` kept (8×). Build + typecheck ✅.

**New component:** `DesignSlider.vue` — Ark UI `Slider`, single-value `number` model (Ark uses a
`number[]` internally, bridged), `min`/`max`/`step`/`disabled`, `ds-focus-ring` on the thumb. Anatomy
matches Ark docs (Root › Control › Track › Range, Thumb[:index=0] › HiddenInput). Not in admin-web.

### 2026-06-23 — quick one-offs migrated

`DevTools.vue` (toast→useDesignToaster, 2 buttons; the popover is now a `gradient-border
bg-surface-raise shadow-floating` card), `ThemeSwitch.vue` (icon button → tertiary), `vault/[id].vue`
(back button + action chips → DesignButton with `border-border-1 border`; 3 UIcon→Icon; utilities→
tokens), `VaultCard.vue` (3 UIcon→Icon; utilities→tokens). No `U*` left in any of the four; build +
typecheck ✅. No new components.

### Next steps (pick up here)

1. **Human visual review** of `/export` (with and without `?id=`) in light + dark. Compare against
   admin-web. Watch the gaps: checkbox styling, select trigger/menu, dialog, toast.
2. If good, **continue Stage 3 sweep** in leverage order. Likely next: a `USkeleton` decision
   (keep vs. build `DesignSkeleton`) since it's the #2 most-used component (26×), and `USwitch`
   (18×) → `DesignSwitch` (already exists in admin-web, not yet ported here).
3. Port remaining admin-web components on demand as pages need them.
4. **Stage 4** (remove `@nuxt/ui`) only once `grep -rE '<U[A-Z]' app/` is empty.

**Components still NOT ported** (port on demand): Switch, Badge, Tooltip, Avatar, Banner,
EmptyState, ErrorState, LoadingState, ViewState, ProgressCircle, DatePicker, RadioGroup,
SegmentGroup, StatusIndicator, TagsInput, Table, Tour, AgeRating, ConfirmProvider. Also no
equivalent yet for: `UInput` (text input — admin-web has `DesignInput`, not yet ported),
`USkeleton`, `UCard`, `UFormField`, `UAlert`, `UStepper`, `USlider`, `UPagination`,
`UNavigationMenu` (used in the layout header), `UContainer`.
