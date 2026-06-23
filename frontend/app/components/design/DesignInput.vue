<script setup lang="ts">
import { Field } from "@ark-ui/vue";

interface Props {
    label?: string;
    placeholder?: string;
    type?: "text" | "email" | "url" | "date" | "time" | "search" | "password";
    disabled?: boolean;
    required?: boolean;
    invalid?: boolean;
    helperText?: string;
    errorText?: string;
    // Extension beyond admin-web: optional leading icon + #trailing slot.
    leadingIcon?: string;
}

withDefaults(defineProps<Props>(), {
    label: undefined,
    placeholder: undefined,
    type: "text",
    helperText: undefined,
    errorText: undefined,
    leadingIcon: undefined,
});

const model = defineModel<string>();
</script>

<template>
    <Field.Root :disabled="disabled" :required="required" :invalid="invalid">
        <Field.Label
            v-if="label"
            class="text-body-3 text-text-muted mb-1 block"
        >
            {{ label }}
        </Field.Label>
        <div class="relative flex items-center">
            <Icon
                v-if="leadingIcon"
                :name="leadingIcon"
                class="text-text-hint pointer-events-none absolute left-3 size-4"
            />
            <Field.Input
                v-model="model"
                :type="type"
                :placeholder="placeholder"
                :class="[
                    'border-border-1 text-body-3 text-text-default placeholder:text-text-hint data-invalid:border-semantic-error ds-focus-ring w-full rounded-xl border py-2 disabled:cursor-not-allowed disabled:opacity-50',
                    leadingIcon ? 'pl-9' : 'pl-3',
                    $slots.trailing ? 'pr-10' : 'pr-3',
                ]"
            />
            <div
                v-if="$slots.trailing"
                class="absolute right-2 flex items-center"
            >
                <slot name="trailing" />
            </div>
        </div>
        <Field.HelperText
            v-if="helperText && !invalid"
            class="text-caption-1 text-text-hint mt-1"
        >
            {{ helperText }}
        </Field.HelperText>
        <Field.ErrorText
            v-if="errorText"
            class="text-caption-1 text-semantic-error mt-1"
        >
            {{ errorText }}
        </Field.ErrorText>
    </Field.Root>
</template>
