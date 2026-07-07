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
}>();

const emit = defineEmits<{
    (
        e: "start-export",
        payload: { vxIds: string[]; selection: ExportSelection },
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

const selectedResCount = computed(
    () => resolutions.filter((r) => r.enabled).length,
);

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

function attemptExport() {
    confirmOpen.value = true;
}

function confirmExport() {
    confirmOpen.value = false;
    startExport();
}

function startExport() {
    emit("start-export", {
        vxIds: exportableAssets.value.map((a) => a.vxId),
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
        <!-- Bulk paste: detect VX-ids from arbitrary text -->
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

        <!-- Assets to export -->
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

        <div class="space-y-6">
            <!-- Alternative actions -->
            <div
                v-if="config.canExportTimedMetadata"
                class="bg-surface-indent space-y-2 rounded-2xl p-4"
            >
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("export.alternativeActions") }}
                </h3>
                <DesignButton
                    variant="secondary"
                    size="small"
                    icon="tabler:file-export"
                    :disabled="submitting"
                    @click="emit('export-timed-metadata')"
                >
                    {{ $t("export.exportTimedMetadata") }}
                </DesignButton>
                <p class="text-text-muted text-xs">
                    {{ $t("export.exportTimedMetadataHint") }}
                </p>
            </div>

            <!-- Destinations -->
            <section class="space-y-2">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("export.destinations") }}
                </h3>
                <div class="flex flex-col gap-2">
                    <DesignCheckbox
                        v-for="d in config.destinations"
                        :key="d"
                        v-model="destChecked[d]"
                    >
                        <template #label>
                            <span class="text-sm">{{
                                destinationName(d)
                            }}</span>
                            <span
                                class="text-text-muted ml-2 font-mono text-xs"
                            >
                                {{ d }}</span
                            >
                        </template>
                    </DesignCheckbox>
                    <p
                        v-if="config.destinations.length === 0"
                        class="text-text-muted text-xs"
                    >
                        {{ $t("export.noDestinations") }}
                    </p>
                </div>
            </section>

            <!-- Audio source -->
            <div class="space-y-1">
                <label class="text-body-3 text-text-muted block">
                    {{ $t("export.audioSource") }}
                </label>
                <DesignSelect
                    v-model="audioSource"
                    :items="config.audioSources"
                />
            </div>

            <!-- Subclips (per-asset; hidden in bulk mode) -->
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
            </section>

            <!-- Language exports -->
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
                <div class="grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2">
                    <DesignCheckbox
                        v-for="l in config.languages"
                        :key="l.code"
                        v-model="langChecked[l.code]"
                    >
                        <template #label>
                            <span class="font-mono text-sm">{{ l.code }}</span>
                            <span class="text-text-muted"> · {{ l.name }}</span>
                        </template>
                    </DesignCheckbox>
                </div>
            </section>

            <!-- Resolutions -->
            <section class="space-y-2">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("export.resolutions")
                    }}<span v-if="aspectRatio"> — {{ aspectRatio }}</span>
                </h3>
                <div class="space-y-2">
                    <div
                        class="text-text-muted grid grid-cols-[7rem_1fr] gap-6 text-xs"
                    >
                        <span></span>
                        <span>{{ $t("export.downloadableHeader") }}</span>
                    </div>
                    <div
                        v-for="r in resolutions"
                        :key="`${r.width}x${r.height}`"
                        class="grid grid-cols-[7rem_1fr] items-center gap-6"
                    >
                        <DesignCheckbox v-model="r.enabled">
                            <template #label>
                                <span class="font-mono text-sm">
                                    {{ r.width }}x{{ r.height }}
                                </span>
                            </template>
                        </DesignCheckbox>
                        <DesignCheckbox
                            v-model="r.downloadable"
                            :disabled="!r.enabled"
                            :aria-label="$t('export.downloadable')"
                        />
                    </div>
                </div>
            </section>

            <!-- Overlay -->
            <div class="space-y-1">
                <label class="text-body-3 text-text-muted block">
                    {{ $t("export.overlay") }}
                </label>
                <DesignSelect v-model="overlay" :items="config.overlays" />
            </div>

            <!-- Options -->
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

        <!-- Sticky action bar -->
        <div
            class="bg-surface-raise gradient-border shadow-floating sticky bottom-6 -mx-6 mt-6 rounded-2xl px-6 py-4"
        >
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

        <!-- Export confirmation (both single-asset and bulk) -->
        <DesignDialog
            v-model:open="confirmOpen"
            :title="confirmTitle"
            :description="confirmMessage"
        >
            <div class="flex w-full justify-end gap-2">
                <DesignButton variant="tertiary" @click="confirmOpen = false">
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
        </DesignDialog>
    </div>
</template>
