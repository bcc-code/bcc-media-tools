<script setup lang="ts">
import type { GetExportConfigResponse } from "~~/src/gen/api/v1/api_pb";
import type { ExportSelection } from "~/components/export/types";
import type { AssetRef } from "~/utils/vxids";

const props = defineProps<{
    config: GetExportConfigResponse;
    submitting?: boolean;
    // Bulk mode: no asset preselected; the user pastes text to detect VX-ids.
    bulkMode?: boolean;
    // Assets to export (single asset, or the resolved bulk list).
    initialAssets?: AssetRef[];
    // Resolves pasted VX-ids to titles (bulk mode only).
    resolveTitles?: (ids: string[]) => Promise<AssetRef[]>;
    // Live progress of the running export (the page owns the run loop).
    progress?: { total: number; done: number };
}>();

const emit = defineEmits<{
    (
        e: "start-export",
        payload: { assets: AssetRef[]; selection: ExportSelection },
    ): void;
    (e: "export-timed-metadata"): void;
}>();

const { t } = useI18n();
const { formatNumber } = useNumberFormat();

/* ----------------------------------------------------------- asset list --- */

const pasteText = ref("");
const resolving = ref(false);
const assets = ref<AssetRef[]>(props.initialAssets ?? []);

// Absorb pasted text into the asset list: extract VX-ids, resolve the ones not
// already listed, append them, then clear the box (so the next paste is a fresh
// batch). Clearing re-fires the watcher with empty text, which is a no-op.
watchDebounced(
    pasteText,
    async (text) => {
        if (!props.bulkMode || !props.resolveTitles) return;
        const ids = extractVXIDs(text);
        if (ids.length === 0) return;
        pasteText.value = "";
        const existing = new Set(assets.value.map((a) => a.vxId));
        const fresh = ids.filter((id) => !existing.has(id));
        if (fresh.length === 0) return;
        resolving.value = true;
        try {
            assets.value = [
                ...assets.value,
                ...(await props.resolveTitles(fresh)),
            ];
        } finally {
            resolving.value = false;
        }
    },
    { debounce: 400 },
);

// Keep the list in sync when the selected asset changes (single mode).
watch(
    () => props.initialAssets,
    (a) => {
        if (props.bulkMode) return;
        assets.value = a ?? [];
    },
);

function removeAsset(vxId: string) {
    assets.value = assets.value.filter((a) => a.vxId !== vxId);
}

/* ------------------------------------------------------------------ state --- */

const destChecked = reactive<Record<string, boolean>>(
    Object.fromEntries(props.config.destinations.map((d) => [d, false])),
);

const audioSource = ref(
    props.config.audioSources.includes(props.config.selectedAudioSource)
        ? props.config.selectedAudioSource
        : (props.config.audioSources[0] ?? ""),
);

const langChecked = reactive<Record<string, boolean>>(
    Object.fromEntries(
        props.config.languages.map((l) => [
            l.code,
            props.config.selectedLanguages.includes(l.code),
        ]),
    ),
);

const resolutions = reactive(
    props.config.resolutions.map((r) => ({
        width: r.width,
        height: r.height,
        enabled: true,
        downloadable: false,
    })),
);

const overlay = ref("None");
const subclipChecked = reactive<Record<string, boolean>>(
    Object.fromEntries(props.config.subclips.map((s) => [s.title, false])),
);

const withChapters = ref(false);
const ignoreSilence = ref(false);
const exportAiSubs = ref(false);

/* --------------------------------------------------------------- computed --- */

const selectedLangCount = computed(
    () => props.config.languages.filter((l) => langChecked[l.code]).length,
);

const selectedDestCount = computed(
    () => props.config.destinations.filter((d) => destChecked[d]).length,
);

const enabledResolutions = computed(() => resolutions.filter((r) => r.enabled));
const selectedResCount = computed(() => enabledResolutions.value.length);

// Assets that actually resolved; "not found" pastes are excluded from export.
const exportableAssets = computed(() =>
    assets.value.filter((a) => a.found !== false),
);

// Footer: reason the export is blocked, or a summary of the selection.
const disabledReason = computed(() => {
    if (exportableAssets.value.length === 0)
        return t("export.selectAssetsHint");
    if (selectedDestCount.value === 0) return t("export.selectDestinationHint");
    return "";
});

const selectionSummary = computed(() =>
    [
        t("export.summaryAssets", {
            n: formatNumber(exportableAssets.value.length),
        }),
        t("export.summaryDestinations", { n: selectedDestCount.value }),
        t("export.summaryLanguages", { n: selectedLangCount.value }),
        t("export.summaryResolutions", { n: selectedResCount.value }),
    ].join(" · "),
);

// Overlay preview: the overlay file itself is the image, served by the backend
// /overlay-preview handler. "None" has no preview.
const base = useRuntimeConfig().public.grpcUrl;
const overlayPreviewUrl = computed(
    () => `${base}/overlay-preview?name=${encodeURIComponent(overlay.value)}`,
);
const overlayPreviewFailed = ref(false);
watch(overlay, () => (overlayPreviewFailed.value = false));

// Aspect ratio of the first resolution, e.g. "16:9".
const aspectRatio = computed(() => {
    const first = resolutions[0];
    if (!first) return "";
    const gcd = (a: number, b: number): number => (b === 0 ? a : gcd(b, a % b));
    const d = gcd(first.width, first.height);
    return d === 0 ? "" : `${first.width / d}:${first.height / d}`;
});

/* ---------------------------------------------------------------- actions --- */

function setLangs(codes: string[]) {
    props.config.languages.forEach(
        (l) => (langChecked[l.code] = codes.includes(l.code)),
    );
}

const toggleLang = (code: string) => (langChecked[code] = !langChecked[code]);

// Filter the language chips by code or name (case-insensitive).
const langFilter = ref("");
const filteredLanguages = computed(() => {
    const q = langFilter.value.trim().toLowerCase();
    if (!q) return props.config.languages;
    return props.config.languages.filter(
        (l) =>
            l.code.toLowerCase().includes(q) ||
            l.name.toLowerCase().includes(q),
    );
});
const selectAllLangs = () =>
    setLangs(props.config.languages.map((l) => l.code));
const clearLangs = () => setLangs([]);
const selectMU1 = () =>
    setLangs(props.config.languages.filter((l) => l.mu1).map((l) => l.code));
const selectMU2 = () =>
    setLangs(props.config.languages.filter((l) => l.mu2).map((l) => l.code));

// Every export is confirmed first — bulk runs can launch many workflows at
// once, and single-asset runs are still irreversible.
const confirmOpen = ref(false);

const confirmTitle = computed(() =>
    props.bulkMode ? t("export.bulkConfirmTitle") : t("export.confirmTitle"),
);

const confirmMessage = computed(() =>
    props.bulkMode
        ? t("export.bulkConfirmMessage", {
              n: formatNumber(exportableAssets.value.length),
              d: selectedDestCount.value,
          })
        : t("export.confirmMessage", { d: selectedDestCount.value }),
);

// Full breakdown of what will be exported, shown in the confirmation dialog so
// an irreversible (and potentially bulk) run can be reviewed before launching.
const confirmRows = computed(() => {
    const rows: { label: string; value: string }[] = [];

    rows.push({
        label: t("export.assets"),
        value:
            props.bulkMode || exportableAssets.value.length !== 1
                ? formatNumber(exportableAssets.value.length)
                : (exportableAssets.value[0]?.title ?? ""),
    });
    rows.push({
        label: t("export.destinations"),
        value: props.config.destinations
            .filter((d) => destChecked[d])
            .map(destinationName)
            .join(", "),
    });
    rows.push({ label: t("export.audioSource"), value: audioSource.value });
    if (selectedLangCount.value > 0)
        rows.push({
            label: t("export.languageExports"),
            value: props.config.languages
                .filter((l) => langChecked[l.code])
                .map((l) => l.code)
                .join(", "),
        });
    const res = resolutions.filter((r) => r.enabled);
    if (res.length > 0)
        rows.push({
            label: t("export.resolutions"),
            value: res
                .map(
                    (r) =>
                        `${r.width}x${r.height}${r.downloadable ? " ↓" : ""}`,
                )
                .join(", "),
        });
    if (overlay.value && overlay.value !== "None")
        rows.push({ label: t("export.overlay"), value: overlay.value });
    if (!props.bulkMode) {
        const subs = props.config.subclips
            .filter((s) => subclipChecked[s.title])
            .map((s) => s.title);
        if (subs.length > 0)
            rows.push({ label: t("export.subclips"), value: subs.join(", ") });
    }
    const opts: string[] = [];
    if (withChapters.value) opts.push(t("export.withChapters"));
    if (ignoreSilence.value) opts.push(t("export.ignoreSilence"));
    if (exportAiSubs.value) opts.push(t("export.exportAiSubsShort"));
    if (opts.length > 0)
        rows.push({ label: t("export.options"), value: opts.join(", ") });

    return rows;
});

function attemptExport() {
    confirmOpen.value = true;
}

function confirmExport() {
    confirmOpen.value = false;
    startExport();
}

function startExport() {
    emit("start-export", {
        assets: exportableAssets.value,
        selection: {
            destinations: props.config.destinations.filter(
                (d) => destChecked[d],
            ),
            audioSource: audioSource.value,
            languages: props.config.languages
                .filter((l) => langChecked[l.code])
                .map((l) => l.code),
            resolutions: resolutions
                .filter((r) => r.enabled)
                .map((r) => ({
                    width: r.width,
                    height: r.height,
                    downloadable: r.downloadable,
                })),
            overlay: overlay.value,
            withChapters: withChapters.value,
            ignoreSilence: ignoreSilence.value,
            exportAiSubs: exportAiSubs.value,
            // Subclips are per-asset; not applicable to bulk export.
            subclips: props.bulkMode
                ? []
                : props.config.subclips
                      .filter((s) => subclipChecked[s.title])
                      .map((s) => s.title),
        },
    });
}
</script>

<template>
    <div class="mx-auto w-full max-w-3xl px-6 py-8">
        <section v-if="bulkMode" class="mb-6 space-y-2">
            <h3 class="text-title-3 text-text-default font-semibold">
                {{ $t("export.bulkTitle") }}
            </h3>
            <p class="text-text-muted text-xs">{{ $t("export.bulkHint") }}</p>
            <DesignTextarea
                v-model="pasteText"
                :rows="4"
                :placeholder="$t('export.bulkPlaceholder')"
            />
        </section>

        <section class="mb-6 space-y-2">
            <div class="flex items-center justify-between gap-2">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("export.assets") }}
                </h3>
                <span class="text-text-muted text-xs">
                    {{
                        $t("export.bulkDetected", {
                            n: formatNumber(assets.length),
                        })
                    }}
                </span>
            </div>
            <ul
                v-if="assets.length > 0"
                class="border-border-1 divide-border-1 divide-y rounded-xl border"
            >
                <li
                    v-for="a in assets"
                    :key="a.vxId"
                    class="flex items-center gap-3 px-3 py-2"
                >
                    <span class="font-mono text-sm">{{ a.vxId }}</span>
                    <span
                        class="truncate text-sm"
                        :class="
                            a.found === false
                                ? 'text-semantic-error'
                                : 'text-text-muted'
                        "
                    >
                        <template v-if="a.found === false">
                            {{ $t("export.assetNotFound") }}
                        </template>
                        <template v-else>{{ a.title }}</template>
                    </span>
                    <DesignButton
                        class="ml-auto"
                        icon="tabler:x"
                        variant="tertiary"
                        intent="danger"
                        size="small"
                        :aria-label="$t('export.remove')"
                        @click="removeAsset(a.vxId)"
                    />
                </li>
            </ul>
            <div v-if="resolving" class="space-y-2">
                <DesignSkeleton class="h-9 w-full" />
            </div>
            <p
                v-else-if="bulkMode && assets.length === 0"
                class="text-text-muted text-xs"
            >
                {{ $t("export.bulkNoIds") }}
            </p>
        </section>

        <section
            v-if="!bulkMode && config.canExportTimedMetadata"
            class="gradient-border bg-surface-raise mb-6 flex flex-col gap-3 rounded-2xl p-4 sm:flex-row sm:items-center sm:justify-between"
        >
            <div class="space-y-1">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("export.alternativeActions") }}
                </h3>
                <p class="text-text-muted text-xs">
                    {{ $t("export.exportTimedMetadataHint") }}
                </p>
            </div>
            <DesignButton
                variant="secondary"
                icon="tabler:file-export"
                :disabled="submitting"
                class="border-border-1 shrink-0 border"
                @click="emit('export-timed-metadata')"
            >
                {{ $t("export.exportTimedMetadata") }}
            </DesignButton>
        </section>

        <div class="space-y-6">
            <section class="space-y-2">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("export.destinations") }}
                </h3>
                <div class="flex flex-col gap-3">
                    <DesignCheckbox
                        v-for="d in config.destinations"
                        :key="d"
                        v-model="destChecked[d]"
                        :label="destinationName(d)"
                    />
                    <p
                        v-if="config.destinations.length === 0"
                        class="text-text-muted text-xs"
                    >
                        {{ $t("export.noDestinations") }}
                    </p>
                </div>
            </section>

            <div class="space-y-1">
                <label class="text-body-3 text-text-muted block">
                    {{ $t("export.audioSource") }}
                </label>
                <DesignSelect
                    v-model="audioSource"
                    :items="config.audioSources"
                />
            </div>

            <section v-if="!bulkMode" class="space-y-2">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("export.subclips") }}
                </h3>
                <p class="text-text-muted text-xs">
                    {{ $t("export.subclipsHint") }}
                </p>
                <div
                    v-if="config.subclips.length > 0"
                    class="flex flex-col gap-2"
                >
                    <DesignCheckbox
                        v-for="s in config.subclips"
                        :key="s.title"
                        v-model="subclipChecked[s.title]"
                        :label="s.title"
                    />
                </div>
                <DesignBanner
                    v-else
                    variant="neutral"
                    icon="tabler:scissors"
                    class="border-border-1 border"
                >
                    {{ $t("export.noSubclips") }}
                </DesignBanner>
            </section>

            <section class="space-y-3">
                <div class="flex flex-wrap items-center justify-between gap-2">
                    <h3 class="text-title-3 text-text-default font-semibold">
                        {{ $t("export.languageExports") }}
                    </h3>
                    <div class="flex items-center gap-2">
                        <span class="text-text-muted text-xs">
                            {{
                                $t("export.nSelected", { n: selectedLangCount })
                            }}
                        </span>
                        <DesignButton
                            variant="secondary"
                            size="small"
                            class="border-border-1 border"
                            @click="selectAllLangs"
                        >
                            {{ $t("export.all") }}
                        </DesignButton>
                        <DesignButton
                            variant="secondary"
                            size="small"
                            class="border-border-1 border"
                            @click="clearLangs"
                        >
                            {{ $t("export.none") }}
                        </DesignButton>
                        <DesignButton
                            variant="secondary"
                            size="small"
                            class="border-border-1 border"
                            @click="selectMU1"
                        >
                            MU1
                        </DesignButton>
                        <DesignButton
                            variant="secondary"
                            size="small"
                            class="border-border-1 border"
                            @click="selectMU2"
                        >
                            MU2
                        </DesignButton>
                    </div>
                </div>
                <DesignInput
                    v-model="langFilter"
                    type="search"
                    leading-icon="tabler:search"
                    :placeholder="$t('export.filterLanguages')"
                />
                <div class="flex flex-wrap gap-2">
                    <button
                        v-for="l in filteredLanguages"
                        :key="l.code"
                        type="button"
                        class="ds-focus-ring shadow-resting inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm"
                        :class="
                            langChecked[l.code]
                                ? 'ring-primary-default bg-primary-default/15 text-text-default ring-2'
                                : 'gradient-border bg-surface-default text-text-default hover:bg-surface-raise'
                        "
                        :aria-pressed="langChecked[l.code]"
                        @click="toggleLang(l.code)"
                    >
                        <Icon
                            :name="
                                langChecked[l.code]
                                    ? 'tabler:check'
                                    : 'tabler:plus'
                            "
                            class="size-3.5 shrink-0"
                            :class="
                                langChecked[l.code]
                                    ? 'text-primary-default'
                                    : 'text-text-muted'
                            "
                        />
                        <span class="font-mono">{{ l.code }}</span>
                        <span class="text-text-muted">· {{ l.name }}</span>
                    </button>
                    <p
                        v-if="filteredLanguages.length === 0"
                        class="text-text-muted text-xs"
                    >
                        {{ $t("export.noLanguageMatches") }}
                    </p>
                </div>
            </section>

            <section class="space-y-2">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("export.resolutions")
                    }}<span v-if="aspectRatio"> — {{ aspectRatio }}</span>
                </h3>
                <div class="flex flex-col gap-2">
                    <div
                        v-for="r in resolutions"
                        :key="`${r.width}x${r.height}`"
                        class="flex items-center gap-4"
                    >
                        <div class="w-32">
                            <DesignCheckbox v-model="r.enabled">
                                <template #label>
                                    <span class="font-mono text-sm">
                                        {{ r.width }}x{{ r.height }}
                                    </span>
                                </template>
                            </DesignCheckbox>
                        </div>
                        <DesignCheckbox
                            v-if="r.enabled"
                            v-model="r.downloadable"
                        >
                            <template #label>
                                <span
                                    class="text-text-muted inline-flex items-center gap-1 text-sm"
                                >
                                    <Icon
                                        name="tabler:download"
                                        class="size-3.5"
                                    />
                                    {{ $t("export.downloadableHeader") }}
                                </span>
                            </template>
                        </DesignCheckbox>
                    </div>
                </div>
            </section>

            <div class="space-y-1">
                <label class="text-body-3 text-text-muted block">
                    {{ $t("export.overlay") }}
                </label>
                <DesignSelect v-model="overlay" :items="config.overlays" />
                <div
                    v-if="overlay && overlay !== 'None'"
                    class="bg-surface-default gradient-border mt-2 aspect-video w-full max-w-md overflow-hidden rounded-xl"
                >
                    <img
                        v-if="!overlayPreviewFailed"
                        :src="overlayPreviewUrl"
                        :alt="overlay"
                        class="h-full w-full object-contain"
                        @error="overlayPreviewFailed = true"
                    />
                    <div
                        v-else
                        class="text-text-hint flex h-full w-full flex-col items-center justify-center gap-1"
                    >
                        <Icon name="tabler:photo-off" class="size-6" />
                        <span class="text-caption-1">
                            {{ $t("export.overlayPreviewUnavailable") }}
                        </span>
                    </div>
                </div>
            </div>

            <section class="flex flex-col gap-3">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("export.options") }}
                </h3>
                <DesignCheckbox
                    v-model="withChapters"
                    :label="$t('export.withChapters')"
                />
                <DesignCheckbox
                    v-model="ignoreSilence"
                    :label="$t('export.ignoreSilence')"
                />
                <DesignCheckbox
                    v-model="exportAiSubs"
                    :label="$t('export.exportAiSubs')"
                />
            </section>
        </div>

        <div
            class="bg-surface-raise gradient-border shadow-floating sticky bottom-6 -mx-6 mt-6 space-y-3 rounded-2xl px-6 py-4"
        >
            <div
                v-if="submitting && (progress?.total ?? 0) > 1"
                class="space-y-1"
            >
                <DesignProgress
                    :model-value="progress?.done ?? 0"
                    :max="progress?.total ?? 1"
                />
                <p class="text-text-muted text-xs tabular-nums">
                    {{
                        $t("export.bulkProgress", {
                            done: formatNumber(progress?.done ?? 0),
                            total: formatNumber(progress?.total ?? 0),
                        })
                    }}
                </p>
            </div>
            <div class="flex items-center justify-between gap-4">
                <p
                    class="text-xs"
                    :class="
                        disabledReason
                            ? 'text-semantic-warning'
                            : 'text-text-muted'
                    "
                >
                    {{ disabledReason || selectionSummary }}
                </p>
                <DesignButton
                    variant="primary"
                    size="large"
                    icon="tabler:file-export"
                    :loading="submitting"
                    :disabled="!!disabledReason"
                    @click="attemptExport"
                >
                    {{
                        bulkMode
                            ? $t("export.bulkStart")
                            : $t("export.startExport")
                    }}
                </DesignButton>
            </div>
        </div>

        <DesignDialog v-model:open="confirmOpen" :title="confirmTitle">
            <div class="space-y-4">
                <p class="text-body-3 text-text-muted">{{ confirmMessage }}</p>
                <dl
                    class="border-border-1 divide-border-1 divide-y rounded-xl border text-sm"
                >
                    <div
                        v-for="row in confirmRows"
                        :key="row.label"
                        class="grid grid-cols-[8rem_1fr] gap-3 px-3 py-2"
                    >
                        <dt class="text-text-muted">{{ row.label }}</dt>
                        <dd class="text-text-default break-words">
                            {{ row.value || "—" }}
                        </dd>
                    </div>
                </dl>
                <div class="flex w-full justify-end gap-2">
                    <DesignButton
                        variant="tertiary"
                        @click="confirmOpen = false"
                    >
                        {{ $t("export.cancel") }}
                    </DesignButton>
                    <DesignButton
                        variant="primary"
                        icon="tabler:file-export"
                        @click="confirmExport"
                    >
                        {{
                            bulkMode
                                ? $t("export.bulkStart")
                                : $t("export.startExport")
                        }}
                    </DesignButton>
                </div>
            </div>
        </DesignDialog>
    </div>
</template>
