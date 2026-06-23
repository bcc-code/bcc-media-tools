<script setup lang="ts">
import { Switch } from "@ark-ui/vue";

interface Props {
    label?: string;
    // Extension beyond admin-web: optional helper description under the label.
    description?: string;
    disabled?: boolean;
}

withDefaults(defineProps<Props>(), {
    label: undefined,
    description: undefined,
});

const model = defineModel<boolean>({ default: false });
</script>

<template>
    <Switch.Root
        v-model:checked="model"
        :disabled="disabled"
        class="inline-flex cursor-pointer items-start gap-2.5 disabled:cursor-not-allowed disabled:opacity-50"
    >
        <Switch.Control
            class="bg-text-default/15 data-[state=checked]:bg-primary-contrast ease-out-expo ds-focus-ring mt-0.5 inline-flex h-6 w-10 shrink-0 items-center rounded-full p-0.5 transition-colors duration-200"
        >
            <Switch.Thumb
                class="shadow-resting ease-out-expo data-[state=checked]:bg-surface-default size-5 rounded-full bg-white transition-transform duration-200 data-[state=checked]:translate-x-4"
            />
        </Switch.Control>
        <div v-if="label || description" class="flex flex-col">
            <Switch.Label
                v-if="label"
                class="text-body-3 text-text-default select-none"
            >
                {{ label }}
            </Switch.Label>
            <span
                v-if="description"
                class="text-caption-1 text-text-hint select-none"
            >
                {{ description }}
            </span>
        </div>
        <Switch.HiddenInput />
    </Switch.Root>
</template>
