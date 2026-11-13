<script setup lang="ts">
// Marks a contributor, or a marker with one, who is under 26 today.
// A class set on it lands on the text: the tooltip has no element of its own.
defineOptions({ inheritAttrs: false });
const { t } = useI18n();
</script>

<template>
    <DesignTooltip :content="t('editorial.u26.hint')">
        <span
            v-bind="$attrs"
            class="text-caption-1 text-text-muted shrink-0 font-medium"
        >
            U26
        </span>
    </DesignTooltip>
</template>
