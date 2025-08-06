<script setup lang="ts">
defineProps<{
    title: string;
}>();
</script>

<template>
    <div
        class="grid-span-1 col-span-full grid grid-cols-subgrid items-baseline gap-4 p-4"
    >
        <h3 class="text-sm leading-0 text-neutral-500 dark:text-neutral-400">
            {{ title }}
        </h3>
        <div class="flex flex-col flex-wrap gap-4">
            <slot />
        </div>
    </div>
</template>
