<script setup lang="ts">
useHead({
    title: "Export",
});

type Trigger = {
    id: string;
    name: string;
    url: string;
    description?: string;
};

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
    <div class="mx-auto flex w-full max-w-2xl flex-col p-4">
        <div class="my-8">
            <h1 class="text-2xl font-bold">Export</h1>
            <p class="text-muted">Select and trigger an export workflow</p>
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
