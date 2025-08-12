<script setup lang="ts">
useHead({
    title: "Export",
});

type Trigger = {
    id: string;
    name: string;
    description?: string;
};

const triggers: Trigger[] = [
    { id: "vx-export", name: "VX Export", description: "Export to VX" },
    { id: "vb-export", name: "VB Export", description: "Export to VB" },
    {
        id: "isilon-export",
        name: "Isilon Export",
        description: "Export to Isilon",
    },
    {
        id: "bulk-shorts-export",
        name: "Bulk Shorts Export",
        description: "Export to Bulk Shorts",
    },
    {
        id: "history",
        name: "Workflow History",
        description: "Workflow History",
    },
];
</script>

<template>
    <div class="mx-auto flex h-screen w-full max-w-2xl flex-col p-4">
        <div class="my-8">
            <h1 class="text-2xl font-bold">Export</h1>
            <p class="text-muted">Select and trigger an export workflow</p>
        </div>
        <div class="flex flex-col gap-4">
            <NuxtLink
                v-for="trigger in triggers"
                :key="trigger.id"
                :to="`/export/${trigger.id}`"
            >
                <UCard>
                    <p>{{ trigger.name }}</p>
                    <p v-if="trigger.description" class="text-dimmed text-sm">
                        {{ trigger.description }}
                    </p>
                </UCard>
            </NuxtLink>
        </div>
    </div>
</template>
