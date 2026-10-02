<script lang="ts" setup>
defineProps<{
    modelValue: boolean;
    /** Destination name, e.g. "BMM" or "BCC Media". */
    target: string;
    disabled?: boolean;
}>();

const emit = defineEmits<{
    "update:modelValue": [boolean];
}>();

const { t } = useI18n();
</script>

<template>
    <button
        type="button"
        role="switch"
        :aria-checked="modelValue"
        :disabled="disabled"
        class="ds-focus-ring text-title-1 ease-out-expo flex w-full cursor-pointer items-center justify-center gap-2 rounded-2xl border px-4 py-4 transition-colors duration-200 active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-50"
        :class="
            modelValue
                ? 'border-semantic-success/40 bg-semantic-success/15 text-semantic-success'
                : 'border-border-1 text-text-muted hover:bg-surface-indent'
        "
        @click="emit('update:modelValue', !modelValue)"
    >
        <Icon
            :name="modelValue ? 'tabler:check' : 'tabler:minus'"
            class="size-5 shrink-0"
        />
        {{
            modelValue
                ? t("editorial.publishOn", { target })
                : t("editorial.publishOff", { target })
        }}
    </button>
</template>
