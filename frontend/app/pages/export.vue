<script setup lang="ts">
import type { ExportSelection } from "~/components/export/types";
import type { AssetRef } from "~/utils/vxids";

useHead({
    title: "Export",
});

const route = useRoute();
const vxId = computed(() => route.query.id?.toString());

const api = useAPI();
const toaster = useDesignToaster();
const { goToJobsAction } = useJobsToast();
const { t } = useI18n();
const { formatNumber } = useNumberFormat();
const { me } = useMe();

// Users with the bulkExport permission get the bulk paste form on the empty
// (no-id) state.
const canBulk = computed(
    () => !!(me.value?.admin || me.value?.export?.bulkExport),
);

const {
    data: config,
    status,
    error,
} = useAsyncData(
    () => `export-config:${vxId.value}`,
    () => api.getExportConfig({ VXID: vxId.value }),
    { watch: [vxId], immediate: !!vxId.value },
);

// Asset-independent config for bulk export (loaded lazily when allowed).
const {
    data: bulkConfig,
    status: bulkStatus,
    error: bulkError,
    execute: loadBulkConfig,
} = useAsyncData(
    "export-bulk-config",
    () => api.getExportConfig({ VXID: "" }),
    {
        immediate: false,
    },
);

watch(
    [canBulk, vxId],
    ([can, id]) => {
        if (can && !id && !bulkConfig.value) loadBulkConfig();
    },
    { immediate: true },
);

// Resolve pasted VX-ids to titles for the bulk asset list.
async function resolveTitles(ids: string[]): Promise<AssetRef[]> {
    const res = await api.resolveAssets({ VXIDs: ids });
    return res.assets.map((a) => ({
        vxId: a.VXID,
        title: a.title,
        found: a.found,
    }));
}

const submitting = ref(false);

async function onStartExport({
    vxIds,
    selection,
}: {
    vxIds: string[];
    selection: ExportSelection;
}) {
    if (vxIds.length === 0) return;
    submitting.value = true;
    const failed: string[] = [];
    try {
        for (const id of vxIds) {
            try {
                await api.startExport({ VXID: id, ...selection });
            } catch {
                failed.push(id);
            }
        }
        const started = vxIds.length - failed.length;
        // Deep-link to the single asset's jobs, or the whole list for a bulk run.
        const jobAction =
            started > 0
                ? goToJobsAction(vxIds.length === 1 ? vxIds[0] : undefined)
                : undefined;
        if (failed.length === 0) {
            toaster.create({
                title: t("export.exportStarted"),
                description: t("export.bulkStartedCount", {
                    n: formatNumber(started),
                }),
                type: "success",
                action: jobAction,
            });
        } else {
            toaster.create({
                title: t("export.exportStarted"),
                description: `${t("export.bulkStartedCount", { n: formatNumber(started) })} · ${t("export.bulkFailedCount", { n: formatNumber(failed.length) })}`,
                type: started === 0 ? "error" : "warning",
                action: jobAction,
            });
        }
    } finally {
        submitting.value = false;
    }
}

async function onExportTimedMetadata() {
    if (!vxId.value) return;
    submitting.value = true;
    try {
        await api.exportTimedMetadata({ VXID: vxId.value });
        toaster.create({
            title: t("export.timedMetadataStarted"),
            type: "success",
            action: goToJobsAction(vxId.value),
        });
    } catch (err) {
        toaster.create({
            title: t("export.exportFailed"),
            description: (err as Error)?.message,
            type: "error",
        });
    } finally {
        submitting.value = false;
    }
}

// External trigger links shown on the empty state (no asset selected).
type Trigger = { id: string; name: string; url: string; description?: string };
const triggers: Trigger[] = [
    {
        id: "sync",
        name: "Audio Sync",
        description: "Adjusts related audio with the specified adjustment",
        url: "https://export.bcc.media/ingest-fix/sync",
    },
    {
        id: "ingest-fix",
        name: "Audio Extraction",
        description: "Extract Audio from MU1 and MU2",
        url: "https://export.bcc.media/ingest-fix",
    },
    {
        id: "bulk-shorts-export",
        name: "Bulk Shorts Export",
        url: "https://export.bcc.media/bulk-shorts-export",
    },
    {
        id: "history",
        name: "Workflow History",
        description: "A list of recent workflow runs",
        url: "https://export.bcc.media/list",
    },
];
</script>

<template>
    <!-- Asset selected: show the VX export form -->
    <div v-if="vxId">
        <ExportForm
            v-if="status === 'success' && config"
            :config="config"
            :initial-assets="[{ vxId: config.VXID, title: config.title }]"
            :submitting="submitting"
            @start-export="onStartExport"
            @export-timed-metadata="onExportTimedMetadata"
        />

        <div
            v-else-if="status === 'error'"
            class="mx-auto flex w-full max-w-3xl flex-col items-center gap-4 p-8"
        >
            <Icon name="tabler:alert-triangle" class="text-text-hint size-10" />
            <p class="text-text-muted text-center">
                {{ error?.message ?? $t("export.loadFailed") }}
            </p>
        </div>

        <div v-else class="mx-auto w-full max-w-3xl space-y-4 p-8">
            <DesignSkeleton class="mx-auto h-8 w-80" />
            <DesignSkeleton class="h-24 w-full" />
            <DesignSkeleton class="h-40 w-full" />
            <DesignSkeleton class="h-64 w-full" />
        </div>
    </div>

    <!-- No asset: bulk export (if permitted) + how to open the tool + links -->
    <div v-else>
        <template v-if="canBulk">
            <ExportForm
                v-if="bulkStatus === 'success' && bulkConfig"
                :config="bulkConfig"
                bulk-mode
                :resolve-titles="resolveTitles"
                :submitting="submitting"
                @start-export="onStartExport"
            />
            <div
                v-else-if="bulkStatus === 'error'"
                class="mx-auto flex w-full max-w-3xl flex-col items-center gap-4 p-8"
            >
                <Icon
                    name="tabler:alert-triangle"
                    class="text-text-hint size-10"
                />
                <p class="text-text-muted text-center">
                    {{ bulkError?.message ?? $t("export.loadFailed") }}
                </p>
            </div>
            <div v-else class="mx-auto w-full max-w-3xl space-y-4 p-8">
                <DesignSkeleton class="h-24 w-full" />
                <DesignSkeleton class="h-40 w-full" />
            </div>
        </template>

        <div class="mx-auto flex w-full max-w-2xl flex-col p-4">
            <div class="my-8">
                <h1 class="text-heading-3 text-text-default">
                    {{ $t("export.title") }}
                </h1>
                <p class="text-text-muted">
                    {{ $t("export.openFromMediabanken") }}
                </p>
            </div>
            <div class="flex flex-col gap-4">
                <NuxtLink
                    v-for="trigger in triggers"
                    :key="trigger.id"
                    :to="trigger.url"
                    external
                >
                    <div
                        class="gradient-border bg-surface-raise shadow-resting hover:bg-surface-indent space-y-1 rounded-2xl p-4"
                    >
                        <div class="flex items-center justify-between gap-2">
                            <p class="text-text-default">{{ trigger.name }}</p>
                            <Icon
                                name="tabler:arrow-right"
                                class="text-text-muted"
                            />
                        </div>
                        <p
                            v-if="trigger.description"
                            class="text-text-hint text-sm"
                        >
                            {{ trigger.description }}
                        </p>
                    </div>
                </NuxtLink>
            </div>
        </div>
    </div>
</template>
