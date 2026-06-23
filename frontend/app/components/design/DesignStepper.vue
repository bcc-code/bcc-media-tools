<script setup lang="ts">
import { Steps } from "@ark-ui/vue";

interface StepItem {
    title: string;
    value: string;
    icon?: string;
}

const props = defineProps<{
    modelValue?: string;
    items: StepItem[];
}>();

// Ark Steps is index-based; the public API stays value-based to match callers.
const currentIndex = computed(() => {
    const i = props.items.findIndex((item) => item.value === props.modelValue);
    return i === -1 ? 0 : i;
});
</script>

<template>
    <Steps.Root :step="currentIndex" :count="items.length">
        <Steps.List class="flex items-center">
            <Steps.Item
                v-for="(item, index) in items"
                :key="item.value"
                :index="index"
                class="flex flex-1 items-center gap-3 last:flex-none"
            >
                <Steps.Trigger disabled class="group flex items-center gap-2">
                    <Steps.Indicator
                        class="bg-surface-indent text-text-hint group-data-[current]:bg-primary-default group-data-[current]:text-on-primary group-data-[complete]:bg-primary-default group-data-[complete]:text-on-primary ease-out-expo flex size-8 shrink-0 items-center justify-center rounded-full text-title-3 transition-colors duration-200"
                    >
                        <Icon
                            v-if="item.icon"
                            :name="item.icon"
                            class="size-4"
                        />
                        <template v-else>{{ index + 1 }}</template>
                    </Steps.Indicator>
                    <span
                        class="text-text-muted group-data-[current]:text-text-default text-title-3 whitespace-nowrap"
                    >
                        {{ item.title }}
                    </span>
                </Steps.Trigger>
                <Steps.Separator
                    v-if="index < items.length - 1"
                    class="bg-border-1 data-[complete]:bg-primary-default h-px flex-1"
                />
            </Steps.Item>
        </Steps.List>
    </Steps.Root>
</template>
