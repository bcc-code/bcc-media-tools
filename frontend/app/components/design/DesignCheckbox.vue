<script setup lang="ts">
import { Checkbox } from "@ark-ui/vue";

/*
 * No checkbox exists in the admin-web design system (it models on/off via
 * DesignSwitch). The export form is checkbox-heavy and checkboxes are the
 * semantically correct control for its multi-select lists, so this is a
 * deliberate, minimal extension built on Ark UI's Checkbox primitive and
 * styled with the shared design tokens.
 */
interface Props {
    label?: string;
    description?: string;
    disabled?: boolean;
    ariaLabel?: string;
}

withDefaults(defineProps<Props>(), {
    label: undefined,
    description: undefined,
    ariaLabel: undefined,
});

const model = defineModel<boolean>({ default: false });
</script>

<template>
    <Checkbox.Root
        v-model:checked="model"
        :disabled="disabled"
        :aria-label="ariaLabel"
        class="inline-grid cursor-pointer grid-cols-[auto_1fr] items-center gap-x-2.5 gap-y-0.5 disabled:cursor-not-allowed disabled:opacity-50"
    >
        <Checkbox.Control
            class="border-border-1 data-[state=checked]:bg-primary-default data-[state=checked]:border-primary-default ds-focus-ring flex size-5 shrink-0 items-center justify-center rounded-md border"
        >
            <Checkbox.Indicator class="flex items-center justify-center">
                <Icon name="tabler:check" class="text-on-primary size-3.5" />
            </Checkbox.Indicator>
        </Checkbox.Control>
        <Checkbox.Label
            v-if="label || $slots.label"
            class="text-body-3 text-text-default select-none"
        >
            <slot name="label">{{ label }}</slot>
        </Checkbox.Label>
        <p
            v-if="description || $slots.description"
            class="text-caption-1 text-text-muted col-start-2 select-none"
        >
            <slot name="description">{{ description }}</slot>
        </p>
        <Checkbox.HiddenInput />
    </Checkbox.Root>
</template>
