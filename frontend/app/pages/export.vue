<script setup lang="ts">
import type { ExportSelection } from "~/components/export/types";

useHead({
    title: "Export",
});

const route = useRoute();
const vxId = computed(() => route.query.id?.toString());

const api = useAPI();
const toast = useToast();
const { t } = useI18n();

const {
    data: config,
    status,
    error,
} = useAsyncData(
    () => `export-config:${vxId.value}`,
    () => api.getExportConfig({ VXID: vxId.value }),
    { watch: [vxId], immediate: !!vxId.value },
);

const submitting = ref(false);

async function onStartExport(payload: ExportSelection) {
    if (!vxId.value) return;
    submitting.value = true;
    try {
        const res = await api.startExport({
            VXID: vxId.value,
            destinations: payload.destinations,
            audioSource: payload.audioSource,
            languages: payload.languages,
            resolutions: payload.resolutions,
            overlay: payload.overlay,
            withChapters: payload.withChapters,
            ignoreSilence: payload.ignoreSilence,
            exportAiSubs: payload.exportAiSubs,
            subclips: payload.subclips,
        });
        toast.add({
            icon: "tabler:check",
            title: t("export.exportStarted"),
            description: t("export.exportStartedCount", {
                n: res.workflowIds.length,
            }),
            color: "success",
        });
    } catch (err) {
        toast.add({
            icon: "tabler:alert-triangle",
            title: t("export.exportFailed"),
            description: (err as Error)?.message,
            color: "error",
        });
    } finally {
        submitting.value = false;
    }
}

async function onExportTimedMetadata() {
    if (!vxId.value) return;
    submitting.value = true;
    try {
        await api.exportTimedMetadata({ VXID: vxId.value });
        toast.add({
            icon: "tabler:check",
            title: t("export.timedMetadataStarted"),
            color: "success",
        });
    } catch (err) {
        toast.add({
            icon: "tabler:alert-triangle",
            title: t("export.exportFailed"),
            description: (err as Error)?.message,
            color: "error",
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
            :submitting="submitting"
            @start-export="onStartExport"
            @export-timed-metadata="onExportTimedMetadata"
        />

        <div
            v-else-if="status === 'error'"
            class="mx-auto flex w-full max-w-3xl flex-col items-center gap-4 p-8"
        >
            <UIcon
                name="tabler:alert-triangle"
                class="text-dimmed size-10"
            />
            <p class="text-muted text-center">
                {{ error?.message ?? $t("export.loadFailed") }}
            </p>
        </div>

        <div v-else class="mx-auto w-full max-w-3xl space-y-4 p-8">
            <USkeleton class="mx-auto h-8 w-80" />
            <USkeleton class="h-24 w-full" />
            <USkeleton class="h-40 w-full" />
            <USkeleton class="h-64 w-full" />
        </div>
    </div>

    <!-- No asset: explain how to open the tool + keep external trigger links -->
    <div v-else class="mx-auto flex w-full max-w-2xl flex-col p-4">
        <div class="my-8">
            <h1 class="text-2xl font-bold">{{ $t("export.title") }}</h1>
            <p class="text-muted">{{ $t("export.openFromMediabanken") }}</p>
        </div>
        <div class="flex flex-col gap-4">
            <NuxtLink
                v-for="trigger in triggers"
                :key="trigger.id"
                :to="trigger.url"
                external
            >
                <UCard>
                    <div class="flex items-center justify-between gap-2">
                        <p>{{ trigger.name }}</p>
                        <Icon name="heroicons:arrow-right" class="text-muted" />
                    </div>
                    <p v-if="trigger.description" class="text-dimmed text-sm">
                        {{ trigger.description }}
                    </p>
                </UCard>
            </NuxtLink>
        </div>
    </div>
</template>
