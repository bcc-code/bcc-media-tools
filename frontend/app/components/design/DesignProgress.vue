<script setup lang="ts">
import { Progress } from "@ark-ui/vue";

const props = withDefaults(defineProps<{ max?: number; status?: boolean }>(), {
    max: 100,
    status: false,
});

const model = defineModel<number>({ default: 0 });

const percent = computed(() =>
    Math.min(100, Math.round((model.value / props.max) * 100)),
);
</script>

<template>
    <Progress.Root v-model="model" :max="max" class="flex flex-col gap-1">
        <Progress.Track
            class="bg-text-default/15 h-2 w-full overflow-hidden rounded-full"
        >
            <Progress.Range
                class="bg-primary-contrast ease-out-expo h-full rounded-full transition-[width] duration-200"
            />
        </Progress.Track>
        <span
            v-if="status"
            class="text-caption-1 text-text-muted self-end tabular-nums"
        >
            {{ percent }}%
        </span>
    </Progress.Root>
</template>
