<script setup lang="ts">
import { Slider } from "@ark-ui/vue";

/*
 * No slider exists in the admin-web design system. Built on Ark UI's Slider
 * primitive, token-styled. Single-value (number) model; Ark works with arrays.
 */
interface Props {
    min?: number;
    max?: number;
    step?: number;
    disabled?: boolean;
}

withDefaults(defineProps<Props>(), {
    min: 0,
    max: 100,
    step: 1,
});

const model = defineModel<number>({ default: 0 });

const arrayModel = computed<number[]>({
    get: () => [model.value],
    set: (v) => {
        model.value = v[0] ?? 0;
    },
});
</script>

<template>
    <Slider.Root
        v-model="arrayModel"
        :min="min"
        :max="max"
        :step="step"
        :disabled="disabled"
        class="w-full disabled:cursor-not-allowed disabled:opacity-50"
    >
        <Slider.Control class="relative flex items-center py-1">
            <Slider.Track class="bg-border-1 h-1.5 flex-1 rounded-full">
                <Slider.Range class="bg-primary-contrast h-full rounded-full" />
            </Slider.Track>
            <Slider.Thumb
                :index="0"
                class="gradient-border shadow-resting ds-focus-ring bg-surface-raise block size-5 cursor-grab rounded-full active:cursor-grabbing"
            >
                <Slider.HiddenInput />
            </Slider.Thumb>
        </Slider.Control>
    </Slider.Root>
</template>
