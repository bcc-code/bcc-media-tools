<script setup lang="ts">
const { data, status } = useFetch("/api/exports");
</script>

<template>
    <div class="mx-auto w-full max-w-7xl p-4">
        <h1 class="my-8 text-2xl font-bold">Workflow History</h1>
        <UTable
            :data="data"
            :loading="status === 'pending'"
            :ui="{ tr: 'cursor-pointer' }"
            @select="console.log"
        >
            <template #status-cell="{ getValue }">
                <UBadge
                    variant="soft"
                    :color="getValue() == 'Running' ? 'info' : 'success'"
                >
                    {{ getValue() }}
                </UBadge>
            </template>
        </UTable>
    </div>
</template>
