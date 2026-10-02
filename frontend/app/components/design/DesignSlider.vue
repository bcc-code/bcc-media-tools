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
    // Extension beyond admin-web: "overlay" is for a slider sitting on media,
    // where the themed track colour disappears against the picture.
    variant?: "default" | "overlay";
}

withDefaults(defineProps<Props>(), {
    min: 0,
    max: 100,
    step: 1,
    variant: "default",
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
            <Slider.Track
                class="h-1.5 flex-1 rounded-full"
                :class="variant === 'overlay' ? 'bg-white/30' : 'bg-border-1'"
            >
                <Slider.Range
                    class="h-full rounded-full"
                    :class="
                        variant === 'overlay'
                            ? 'bg-white'
                            : 'bg-primary-contrast'
                    "
                />
            </Slider.Track>
            <Slider.Thumb
                :index="0"
                class="gradient-border shadow-resting ds-focus-ring block size-5 cursor-grab rounded-full active:cursor-grabbing"
                :class="variant === 'overlay' ? 'bg-white' : 'bg-surface-raise'"
            >
                <Slider.HiddenInput />
            </Slider.Thumb>
        </Slider.Control>
    </Slider.Root>
</template>
