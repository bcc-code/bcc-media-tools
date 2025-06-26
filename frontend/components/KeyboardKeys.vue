<script setup lang="ts">
defineProps<{
    keys: string[];
}>();
</script>

<template>
    <div class="flex gap-1">
        <kbd
            v-for="(key, i) in keys"
            :key="key + i"
            class="rounded border border-b-4 border-black bg-white px-2 py-0.5 text-sm shadow-sm"
        >
            {{ key }}
        </kbd>
    </div>
</template>
