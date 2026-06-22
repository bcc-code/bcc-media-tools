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
    Object.fromEntries(props.config.destinations.map((d) => [d, false])),
);

const subtitleShape = ref(props.config.subtitleShapes[0] ?? "None");
const subtitleStyle = ref(props.config.subtitleStyles[0] ?? "");

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
    () => props.config.destinations.filter((d) => destChecked[d]).length,
);

const { t } = useI18n();

// Footer: reason the export is blocked, or a summary of the selection.
const disabledReason = computed(() => {
    if (assets.value.length === 0) return t("vbExport.selectAssetsHint");
    if (selectedDestCount.value === 0)
        return t("vbExport.selectDestinationHint");
    return "";
});

const selectionSummary = computed(() =>
    [
        t("vbExport.summaryAssets", { n: assets.value.length }),
        t("vbExport.summaryDestinations", { n: selectedDestCount.value }),
    ].join(" · "),
);

/* ---------------------------------------------------------------- actions --- */

function startExport() {
    emit("start-export", {
        vxIds: assets.value.map((a) => a.vxId),
        selection: {
            destinations: props.config.destinations.filter(
                (d) => destChecked[d],
            ),
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
            <h3 class="text-highlighted text-sm font-semibold">
                {{ $t("vbExport.bulkTitle") }}
            </h3>
            <p class="text-muted text-xs">{{ $t("vbExport.bulkHint") }}</p>
            <UTextarea
                v-model="pasteText"
                :rows="4"
                autoresize
                :placeholder="$t('vbExport.bulkPlaceholder')"
                class="w-full"
            />
        </section>

        <!-- Assets to export -->
        <section class="mb-6 space-y-2">
            <div class="flex items-center justify-between gap-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("vbExport.assets") }}
                </h3>
                <span class="text-muted text-xs">
                    {{ $t("vbExport.bulkDetected", { n: assets.length }) }}
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
                            {{ $t("vbExport.assetNotFound") }}
                        </template>
                        <template v-else>{{ a.title }}</template>
                    </span>
                    <UButton
                        class="ml-auto"
                        icon="tabler:x"
                        color="neutral"
                        variant="ghost"
                        size="xs"
                        :aria-label="$t('vbExport.remove')"
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
                {{ $t("vbExport.bulkNoIds") }}
            </p>
        </section>

        <div class="space-y-6">
            <!-- Destinations -->
            <section class="space-y-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("vbExport.destinations") }}
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
                        {{ $t("vbExport.noDestinations") }}
                    </p>
                </div>
            </section>

            <!-- Subtitles (burn-in) -->
            <UFormField :label="$t('vbExport.subtitlesBurnIn')">
                <USelect
                    v-model="subtitleShape"
                    :items="config.subtitleShapes"
                    class="w-full"
                />
            </UFormField>

            <!-- Subtitles burn in style -->
            <UFormField :label="$t('vbExport.subtitlesBurnInStyle')">
                <USelect
                    v-model="subtitleStyle"
                    :items="config.subtitleStyles"
                    class="w-full"
                />
            </UFormField>
        </div>

        <!-- Sticky action bar -->
        <div
            class="bg-default border-default sticky bottom-0 -mx-6 mt-6 rounded-2xl border px-6 py-4"
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
                            ? $t("vbExport.bulkStart")
                            : $t("vbExport.startExport")
                    }}
                </UButton>
            </div>
        </div>
    </div>
</template>
