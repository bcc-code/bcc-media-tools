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

// Footer: reason the export is blocked, or a summary of the selection.
const disabledReason = computed(() => {
    if (assets.value.length === 0) return t("export.selectAssetsHint");
    if (selectedDestCount.value === 0) return t("export.selectDestinationHint");
    return "";
});

const selectionSummary = computed(() =>
    [
        t("export.summaryAssets", { n: assets.value.length }),
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

function startExport() {
    emit("start-export", {
        vxIds: assets.value.map((a) => a.vxId),
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
            <h3 class="text-highlighted text-sm font-semibold">
                {{ $t("export.bulkTitle") }}
            </h3>
            <p class="text-muted text-xs">{{ $t("export.bulkHint") }}</p>
            <UTextarea
                v-model="pasteText"
                :rows="4"
                autoresize
                :placeholder="$t('export.bulkPlaceholder')"
                class="w-full"
            />
        </section>

        <!-- Assets to export -->
        <section class="mb-6 space-y-2">
            <div class="flex items-center justify-between gap-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.assets") }}
                </h3>
                <span class="text-muted text-xs">
                    {{ $t("export.bulkDetected", { n: assets.length }) }}
                </span>
            </div>
            <ul
                v-if="assets.length > 0"
                class="border-default divide-default divide-y rounded-md border"
            >
                <li
                    v-for="a in assets"
                    :key="a.vxId"
                    class="flex items-center gap-3 px-3 py-2"
                >
                    <span class="font-mono text-sm">{{ a.vxId }}</span>
                    <span class="text-muted truncate text-sm">
                        <template v-if="a.found === false">
                            {{ $t("export.assetNotFound") }}
                        </template>
                        <template v-else>{{ a.title }}</template>
                    </span>
                    <UButton
                        class="ml-auto"
                        icon="tabler:x"
                        color="neutral"
                        variant="ghost"
                        size="xs"
                        :aria-label="$t('export.remove')"
                        @click="removeAsset(a.vxId)"
                    />
                </li>
            </ul>
            <div v-if="resolving" class="space-y-2">
                <USkeleton class="h-9 w-full" />
            </div>
            <p
                v-else-if="bulkMode && assets.length === 0"
                class="text-muted text-xs"
            >
                {{ $t("export.bulkNoIds") }}
            </p>
        </section>

        <div class="space-y-6">
            <!-- Alternative actions -->
            <UCard
                v-if="config.canExportTimedMetadata"
                variant="subtle"
                :ui="{ body: 'space-y-2' }"
            >
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.alternativeActions") }}
                </h3>
                <UButton
                    color="neutral"
                    variant="outline"
                    size="sm"
                    icon="tabler:file-export"
                    :disabled="submitting"
                    @click="emit('export-timed-metadata')"
                >
                    {{ $t("export.exportTimedMetadata") }}
                </UButton>
                <p class="text-muted text-xs">
                    {{ $t("export.exportTimedMetadataHint") }}
                </p>
            </UCard>

            <!-- Destinations -->
            <section class="space-y-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.destinations") }}
                </h3>
                <div class="flex flex-col gap-2">
                    <UCheckbox
                        v-for="d in config.destinations"
                        :key="d"
                        v-model="destChecked[d]"
                    >
                        <template #label>
                            <span class="text-sm">{{
                                destinationName(d)
                            }}</span>
                            <span class="text-muted ml-2 font-mono text-xs">
                                {{ d }}</span
                            >
                        </template>
                    </UCheckbox>
                    <p
                        v-if="config.destinations.length === 0"
                        class="text-muted text-xs"
                    >
                        {{ $t("export.noDestinations") }}
                    </p>
                </div>
            </section>

            <!-- Audio source -->
            <UFormField :label="$t('export.audioSource')">
                <USelect
                    v-model="audioSource"
                    :items="config.audioSources"
                    class="w-full"
                />
            </UFormField>

            <!-- Subclips (per-asset; hidden in bulk mode) -->
            <section v-if="!bulkMode" class="space-y-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.subclips") }}
                </h3>
                <p class="text-muted text-xs">
                    {{ $t("export.subclipsHint") }}
                </p>
                <div
                    v-if="config.subclips.length > 0"
                    class="flex flex-col gap-2"
                >
                    <UCheckbox
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
                    <h3 class="text-highlighted text-sm font-semibold">
                        {{ $t("export.languageExports") }}
                    </h3>
                    <div class="flex items-center gap-2">
                        <span class="text-muted text-xs">
                            {{
                                $t("export.nSelected", { n: selectedLangCount })
                            }}
                        </span>
                        <UButton
                            color="neutral"
                            variant="subtle"
                            size="xs"
                            @click="selectAllLangs"
                        >
                            {{ $t("export.all") }}
                        </UButton>
                        <UButton
                            color="neutral"
                            variant="subtle"
                            size="xs"
                            @click="clearLangs"
                        >
                            {{ $t("export.none") }}
                        </UButton>
                        <UButton
                            color="neutral"
                            variant="subtle"
                            size="xs"
                            @click="selectMU1"
                        >
                            MU1
                        </UButton>
                        <UButton
                            color="neutral"
                            variant="subtle"
                            size="xs"
                            @click="selectMU2"
                        >
                            MU2
                        </UButton>
                    </div>
                </div>
                <div class="grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2">
                    <UCheckbox
                        v-for="l in config.languages"
                        :key="l.code"
                        v-model="langChecked[l.code]"
                    >
                        <template #label>
                            <span class="font-mono text-sm">{{ l.code }}</span>
                            <span class="text-muted"> · {{ l.name }}</span>
                        </template>
                    </UCheckbox>
                </div>
            </section>

            <!-- Resolutions -->
            <section class="space-y-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.resolutions")
                    }}<span v-if="aspectRatio"> — {{ aspectRatio }}</span>
                </h3>
                <div class="space-y-2">
                    <div
                        class="text-muted grid grid-cols-[7rem_1fr] gap-6 text-xs"
                    >
                        <span></span>
                        <span>{{ $t("export.downloadableHeader") }}</span>
                    </div>
                    <div
                        v-for="r in resolutions"
                        :key="`${r.width}x${r.height}`"
                        class="grid grid-cols-[7rem_1fr] items-center gap-6"
                    >
                        <UCheckbox v-model="r.enabled">
                            <template #label>
                                <span class="font-mono text-sm">
                                    {{ r.width }}x{{ r.height }}
                                </span>
                            </template>
                        </UCheckbox>
                        <UCheckbox
                            v-model="r.downloadable"
                            :disabled="!r.enabled"
                            :aria-label="$t('export.downloadable')"
                        />
                    </div>
                </div>
            </section>

            <!-- Overlay -->
            <UFormField :label="$t('export.overlay')">
                <USelect
                    v-model="overlay"
                    :items="config.overlays"
                    class="w-full"
                />
            </UFormField>

            <!-- Options -->
            <section class="space-y-3">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.options") }}
                </h3>
                <UCheckbox
                    v-model="withChapters"
                    :label="$t('export.withChapters')"
                />
                <UCheckbox
                    v-model="ignoreSilence"
                    :label="$t('export.ignoreSilence')"
                />
                <UCheckbox
                    v-model="exportAiSubs"
                    :label="$t('export.exportAiSubs')"
                />
            </section>
        </div>

        <!-- Sticky action bar -->
        <div
            class="bg-default border-default sticky bottom-6 -mx-6 mt-6 rounded-2xl border px-6 py-4"
        >
            <div class="flex items-center justify-between gap-4">
                <p
                    class="text-xs"
                    :class="disabledReason ? 'text-warning' : 'text-muted'"
                >
                    {{ disabledReason || selectionSummary }}
                </p>
                <UButton
                    size="lg"
                    icon="tabler:file-export"
                    :loading="submitting"
                    :disabled="!!disabledReason"
                    @click="startExport"
                >
                    {{
                        bulkMode
                            ? $t("export.bulkStart")
                            : $t("export.startExport")
                    }}
                </UButton>
            </div>
        </div>
    </div>
</template>
