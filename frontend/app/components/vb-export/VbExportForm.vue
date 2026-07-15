<script setup lang="ts">
import type { GetVBExportConfigResponse } from "~~/src/gen/api/v1/api_pb";
import type { VBExportSelection } from "~/components/vb-export/types";
import type { AssetRef } from "~/utils/vxids";

const props = defineProps<{
    config: GetVBExportConfigResponse;
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
        payload: { vxIds: string[]; selection: VBExportSelection },
    ): void;
}>();

/* ------------------------------------------------------------------ state --- */

const destChecked = reactive<Record<string, boolean>>(
    Object.fromEntries(props.config.destinations.map((d) => [d.id, false])),
);

const subtitleShape = ref(props.config.subtitleShapes[0] ?? "None");
const subtitleStyle = ref(props.config.subtitleStyles[0] ?? "");

// Backend base URL, where the /subtitle-style-preview handler lives.
const base = useRuntimeConfig().public.grpcUrl;

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

/* --------------------------------------------------------------- computed --- */

const selectedDestCount = computed(
    () => props.config.destinations.filter((d) => destChecked[d.id]).length,
);

const { t } = useI18n();
const { formatNumber } = useNumberFormat();

// Assets that actually resolved; "not found" pastes are excluded from export.
const exportableAssets = computed(() =>
    assets.value.filter((a) => a.found !== false),
);

// Footer: reason the export is blocked, or a summary of the selection.
const disabledReason = computed(() => {
    if (exportableAssets.value.length === 0)
        return t("vbExport.selectAssetsHint");
    if (selectedDestCount.value === 0)
        return t("vbExport.selectDestinationHint");
    return "";
});

const selectionSummary = computed(() =>
    [
        t("vbExport.summaryAssets", {
            n: formatNumber(exportableAssets.value.length),
        }),
        t("vbExport.summaryDestinations", { n: selectedDestCount.value }),
    ].join(" · "),
);

/* ---------------------------------------------------------------- actions --- */

// Every export is confirmed first — bulk runs can launch many workflows at
// once, and single-asset runs are still irreversible.
const confirmOpen = ref(false);

const confirmTitle = computed(() =>
    props.bulkMode
        ? t("vbExport.bulkConfirmTitle")
        : t("vbExport.confirmTitle"),
);

const confirmMessage = computed(() =>
    props.bulkMode
        ? t("vbExport.bulkConfirmMessage", {
              n: formatNumber(exportableAssets.value.length),
              d: selectedDestCount.value,
          })
        : t("vbExport.confirmMessage", { d: selectedDestCount.value }),
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
            destinations: props.config.destinations
                .filter((d) => destChecked[d.id])
                .map((d) => d.id),
            subtitleShape: subtitleShape.value,
            subtitleStyle: subtitleStyle.value,
        },
    });
}
</script>

<template>
    <div class="mx-auto w-full max-w-3xl px-6 py-8">
        <!-- Bulk paste: detect VX-ids from arbitrary text -->
        <section v-if="bulkMode" class="mb-6 space-y-2">
            <h3 class="text-title-3 text-text-default font-semibold">
                {{ $t("vbExport.bulkTitle") }}
            </h3>
            <p class="text-text-muted text-xs">{{ $t("vbExport.bulkHint") }}</p>
            <DesignTextarea
                v-model="pasteText"
                :rows="4"
                :placeholder="$t('vbExport.bulkPlaceholder')"
            />
        </section>

        <!-- Assets to export -->
        <section class="mb-6 space-y-2">
            <div class="flex items-center justify-between gap-2">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("vbExport.assets") }}
                </h3>
                <span class="text-text-muted text-xs">
                    {{
                        $t("vbExport.bulkDetected", {
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
                            {{ $t("vbExport.assetNotFound") }}
                        </template>
                        <template v-else>{{ a.title }}</template>
                    </span>
                    <DesignButton
                        class="ml-auto"
                        icon="tabler:x"
                        variant="tertiary"
                        intent="danger"
                        size="small"
                        :aria-label="$t('vbExport.remove')"
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
                {{ $t("vbExport.bulkNoIds") }}
            </p>
        </section>

        <div class="space-y-6">
            <!-- Destinations -->
            <section class="space-y-2">
                <h3 class="text-title-3 text-text-default font-semibold">
                    {{ $t("vbExport.destinations") }}
                </h3>
                <div class="flex flex-col gap-3">
                    <DesignCheckbox
                        v-for="d in config.destinations"
                        :key="d.id"
                        v-model="destChecked[d.id]"
                        :label="destinationName(d.id)"
                        :description="d.description"
                    />
                    <p
                        v-if="config.destinations.length === 0"
                        class="text-text-muted text-xs"
                    >
                        {{ $t("vbExport.noDestinations") }}
                    </p>
                </div>
            </section>

            <!-- Subtitle burn-in — per-asset, so not configurable in bulk mode -->
            <DesignBanner
                v-if="bulkMode"
                variant="info"
                icon="tabler:info-circle"
            >
                {{ $t("vbExport.subtitlesBurnInBulkNote") }}
            </DesignBanner>
            <template v-else>
                <!-- Subtitles (burn-in) -->
                <div class="space-y-1">
                    <label class="text-body-3 text-text-muted block">
                        {{ $t("vbExport.subtitlesBurnIn") }}
                    </label>
                    <DesignSelect
                        v-model="subtitleShape"
                        :items="config.subtitleShapes"
                    />
                </div>

                <!-- Subtitles burn in style -->
                <div class="space-y-2">
                    <label class="text-body-3 text-text-muted block">
                        {{ $t("vbExport.subtitlesBurnInStyle") }}
                    </label>
                    <VbExportSubtitleStylePicker
                        v-model="subtitleStyle"
                        :styles="config.subtitleStyles"
                        :base="base"
                    />
                </div>
            </template>
        </div>

        <!-- Sticky action bar -->
        <div
            class="bg-surface-default border-border-1 sticky bottom-0 -mx-6 mt-6 rounded-2xl border px-6 py-4"
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
                            ? $t("vbExport.bulkStart")
                            : $t("vbExport.startExport")
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
                    {{ $t("vbExport.cancel") }}
                </DesignButton>
                <DesignButton
                    variant="primary"
                    icon="tabler:file-export"
                    @click="confirmExport"
                >
                    {{
                        bulkMode
                            ? $t("vbExport.bulkStart")
                            : $t("vbExport.startExport")
                    }}
                </DesignButton>
            </div>
        </DesignDialog>
    </div>
</template>
