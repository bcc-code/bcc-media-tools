<script setup lang="ts">
import type { VBExportSelection } from "~/components/vb-export/types";

useHead({
    title: "VB Export",
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
    () => `vb-export-config:${vxId.value}`,
    () => api.getVBExportConfig({ VXID: vxId.value }),
    { watch: [vxId], immediate: !!vxId.value },
);

const submitting = ref(false);

async function onStartExport(payload: VBExportSelection) {
    if (!vxId.value) return;
    submitting.value = true;
    try {
        await api.startVBExport({
            VXID: vxId.value,
            destinations: payload.destinations,
            subtitleShape: payload.subtitleShape,
            subtitleStyle: payload.subtitleStyle,
        });
        toast.add({
            icon: "tabler:check",
            title: t("vbExport.exportStarted"),
            color: "success",
        });
    } catch (err) {
        toast.add({
            icon: "tabler:alert-triangle",
            title: t("vbExport.exportFailed"),
            description: (err as Error)?.message,
            color: "error",
        });
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
            :submitting="submitting"
            @start-export="onStartExport"
        />

        <div
            v-else-if="status === 'error'"
            class="mx-auto flex w-full max-w-3xl flex-col items-center gap-4 p-8"
        >
            <UIcon name="tabler:alert-triangle" class="text-dimmed size-10" />
            <p class="text-muted text-center">
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

    <div v-else class="mx-auto flex w-full max-w-2xl flex-col p-4">
        <div class="my-8">
            <h1 class="text-2xl font-bold">{{ $t("vbExport.title") }}</h1>
            <p class="text-muted">{{ $t("vbExport.openFromMediabanken") }}</p>
        </div>
    </div>
</template>
