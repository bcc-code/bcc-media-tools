<script setup lang="ts">
import type { VBExportSelection } from "~/components/vb-export/types";
import type { AssetRef } from "~/utils/vxids";

useHead({
    title: "VB Export",
});

const route = useRoute();
const vxId = computed(() => route.query.id?.toString());

const api = useAPI();
const toaster = useDesignToaster();
const { t } = useI18n();
const { me } = useMe();

// Users with the bulkExport permission get the bulk paste form on the empty
// (no-id) state.
const canBulk = computed(
    () => !!(me.value?.admin || me.value?.vbExport?.bulkExport),
);

const {
    data: config,
    status,
    error,
} = useAsyncData(
    () => `vb-export-config:${vxId.value}`,
    () => api.getVBExportConfig({ VXID: vxId.value }),
    { watch: [vxId], immediate: !!vxId.value },
);

// Asset-independent config for bulk export (loaded lazily when allowed).
const {
    data: bulkConfig,
    status: bulkStatus,
    error: bulkError,
    execute: loadBulkConfig,
} = useAsyncData(
    "vb-export-bulk-config",
    () => api.getVBExportConfig({ VXID: "" }),
    { immediate: false },
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
    selection: VBExportSelection;
}) {
    if (vxIds.length === 0) return;
    submitting.value = true;
    const failed: string[] = [];
    try {
        for (const id of vxIds) {
            try {
                await api.startVBExport({ VXID: id, ...selection });
            } catch {
                failed.push(id);
            }
        }
        const started = vxIds.length - failed.length;
        if (failed.length === 0) {
            toaster.create({
                title: t("vbExport.exportStarted"),
                description: t("vbExport.bulkStartedCount", { n: started }),
                type: "success",
            });
        } else {
            toaster.create({
                title: t("vbExport.exportStarted"),
                description: `${t("vbExport.bulkStartedCount", { n: started })} · ${t("vbExport.bulkFailedCount", { n: failed.length })}`,
                type: started === 0 ? "error" : "warning",
            });
        }
    } finally {
        submitting.value = false;
    }
}
</script>

<template>
    <div v-if="vxId">
        <VbExportForm
            v-if="status === 'success' && config"
            :config="config"
            :initial-assets="[{ vxId: config.VXID, title: config.title }]"
            :submitting="submitting"
            @start-export="onStartExport"
        />

        <div
            v-else-if="status === 'error'"
            class="mx-auto flex w-full max-w-3xl flex-col items-center gap-4 p-8"
        >
            <Icon name="tabler:alert-triangle" class="text-text-hint size-10" />
            <p class="text-text-muted text-center">
                {{ error?.message ?? $t("vbExport.loadFailed") }}
            </p>
        </div>

        <div v-else class="mx-auto w-full max-w-3xl space-y-4 p-8">
            <USkeleton class="mx-auto h-8 w-80" />
            <USkeleton class="h-40 w-full" />
            <USkeleton class="h-12 w-full" />
            <USkeleton class="h-12 w-full" />
        </div>
    </div>

    <div v-else>
        <template v-if="canBulk">
            <VbExportForm
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
                    {{ bulkError?.message ?? $t("vbExport.loadFailed") }}
                </p>
            </div>
            <div v-else class="mx-auto w-full max-w-3xl space-y-4 p-8">
                <USkeleton class="h-24 w-full" />
                <USkeleton class="h-40 w-full" />
            </div>
        </template>

        <div class="mx-auto flex w-full max-w-2xl flex-col p-4">
            <div class="my-8">
                <h1 class="text-heading-3 text-text-default">
                    {{ $t("vbExport.title") }}
                </h1>
                <p class="text-text-muted">
                    {{ $t("vbExport.openFromMediabanken") }}
                </p>
            </div>
        </div>
    </div>
</template>
