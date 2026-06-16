<script setup lang="ts">
import { CantemoAction } from "~~/src/gen/api/v1/api_pb";

definePageMeta({ layout: "iframe" });

const route = useRoute();
const vxId = computed(() => route.query.id?.toString());

const { me } = useMe();
const api = useAPI();
const toast = useToast();

useHead({
    title: "Actions",
    htmlAttrs: { style: "background:#2f2f38;" },
    bodyAttrs: { style: "background:#2f2f38; margin:0;" },
    link: [
        { rel: "preconnect", href: "https://fonts.googleapis.com" },
        {
            rel: "preconnect",
            href: "https://fonts.gstatic.com",
            crossorigin: "",
        },
        {
            rel: "stylesheet",
            href: "https://fonts.googleapis.com/css2?family=Asap:wght@400;500;600;700&display=swap",
        },
    ],
});

type Chip = {
    name: string;
    action: string;
    color: string;
    enabled: boolean;
    run: () => void | Promise<void>;
};

// Name of the chip whose workflow is currently being triggered (disables it).
const loading = ref<string | null>(null);

async function trigger(name: string, action: CantemoAction, started: string) {
    if (!vxId.value || loading.value) return;
    loading.value = name;
    try {
        await api.triggerCantemoAction({ VXID: vxId.value, action });
        toast.add({ icon: "tabler:check", title: started, color: "success" });
    } catch (err) {
        toast.add({
            icon: "tabler:alert-triangle",
            title: "Failed to start",
            description: (err as Error)?.message,
            color: "error",
        });
    } finally {
        loading.value = null;
    }
}

// The panel is embedded as a cross-origin iframe, so navigation chips open the
// export tools in a new tab.
function openTool(path: string) {
    if (!vxId.value) return;
    window.open(`${path}?id=${vxId.value}`, "_blank");
}

const chips = computed<Chip[]>(() => {
    const m = me.value;
    return [
        {
            name: "Export",
            action: "Go to VX export",
            color: "#9aa0a8",
            enabled: !!(
                m?.admin ||
                (m?.export &&
                    (m.export.destinations.length > 0 ||
                        m.export.admin ||
                        m.export.timedMetadata))
            ),
            run: () => openTool("/export/"),
        },
        {
            name: "Export Oslofjord",
            action: "Go to VB export",
            color: "#3c61d8",
            enabled: !!(
                m?.admin ||
                (m?.vbExport &&
                    (m.vbExport.destinations.length > 0 || m.vbExport.admin))
            ),
            run: () => openTool("/vb-export/"),
        },
        {
            name: "Make preview",
            action: "Trigger preview generation",
            color: "#cdbf3a",
            enabled: !!(m?.admin || m?.cantemo?.preview),
            run: () =>
                trigger(
                    "Make preview",
                    CantemoAction.PREVIEW,
                    "Preview generation started",
                ),
        },
        {
            name: "Transcribe",
            action: "Trigger transcription",
            color: "#3fb84f",
            enabled: !!(m?.admin || m?.cantemo?.transcribe),
            run: () =>
                trigger(
                    "Transcribe",
                    CantemoAction.TRANSCRIBE,
                    "Transcription started",
                ),
        },
        {
            name: "Update subtitle from Subtrans",
            action: "Trigger appropriate workflow",
            color: "#3fb84f",
            enabled: !!(m?.admin || m?.cantemo?.subtitles),
            run: () =>
                trigger(
                    "Update subtitle from Subtrans",
                    CantemoAction.SUBTITLE_FROM_SUBTRANS,
                    "Subtitle update started",
                ),
        },
        {
            name: "Update asset relations",
            action: "Update asset relations flow",
            color: "#3c61d8",
            enabled: !!(m?.admin || m?.cantemo?.relations),
            run: () =>
                trigger(
                    "Update asset relations",
                    CantemoAction.UPDATE_RELATIONS,
                    "Asset relations update started",
                ),
        },
    ].filter((c) => c.enabled);
});
</script>

<template>
    <div v-if="vxId" class="cantemo-panel">
        <div class="cantemo-chips">
            <button
                v-for="chip in chips"
                :key="chip.name"
                :title="chip.action"
                class="cantemo-chip"
                :disabled="loading === chip.name"
                @click="chip.run()"
            >
                <span class="cantemo-dot" :style="{ background: chip.color }" />
                {{ chip.name }}
            </button>
        </div>
    </div>
</template>

<style scoped>
.cantemo-panel {
    background: #2f2f38;
    font-family: "Asap", system-ui, sans-serif;
    padding: 22px 24px;
    -webkit-font-smoothing: antialiased;
}

.cantemo-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 9px;
}

.cantemo-chip {
    display: inline-flex;
    align-items: center;
    gap: 9px;
    padding: 8px 15px 8px 13px;
    border: 0;
    border-radius: 5px;
    background: rgba(255, 255, 255, 0.045);
    color: #c7cad0;
    font-family: inherit;
    font-size: 12.5px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
    transition:
        background 0.12s,
        color 0.12s;
}

.cantemo-chip:hover {
    background: rgba(255, 255, 255, 0.11);
    color: #eef0f3;
}

.cantemo-chip:disabled {
    cursor: default;
    opacity: 0.6;
}

.cantemo-dot {
    width: 6px;
    height: 6px;
    flex: 0 0 6px;
    border-radius: 50%;
}
</style>
