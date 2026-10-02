<script lang="ts" setup>
defineProps<{
    modelValue: boolean;
    /** Destination name, e.g. "BMM" or "BCC Media". */
    target: string;
    /** Destination app icon, served from /public. */
    logo: string;
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
        class="ds-focus-ring text-title-1 ease-out-expo flex w-full cursor-pointer items-center gap-3 rounded-2xl border px-4 py-3 transition-colors duration-200 active:scale-[0.99]"
        :class="
            modelValue
                ? 'border-semantic-success/40 bg-semantic-success/15 text-semantic-success'
                : 'border-border-1 text-text-muted hover:bg-surface-indent'
        "
        @click="emit('update:modelValue', !modelValue)"
    >
        <!-- Decorative: the label names the destination. Greyed out while the
             item is not going there, so the row reads at a glance. -->
        <img
            :src="logo"
            alt=""
            class="size-8 shrink-0 rounded-lg transition duration-200"
            :class="modelValue ? '' : 'opacity-40 grayscale'"
        />
        <span class="flex-1 text-left">
            {{
                modelValue
                    ? t("editorial.publishOn", { target })
                    : t("editorial.publishOff", { target })
            }}
        </span>
        <Icon
            :name="modelValue ? 'tabler:check' : 'tabler:minus'"
            class="size-5 shrink-0"
        />
    </button>
</template>
