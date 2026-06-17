<script setup lang="ts">
definePageMeta({ layout: "iframe" });

const route = useRoute();
const vxId = computed(() => route.query.id?.toString());

const { chips, loading } = useCantemoActions(vxId);

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
