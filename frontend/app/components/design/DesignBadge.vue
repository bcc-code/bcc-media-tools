<script setup lang="ts">
import { cva } from "cva";

interface Props {
    label?: string;
    variant?: "success" | "warning" | "info" | "error" | "neutral";
    // Extension beyond admin-web: a larger badge for status at a glance.
    size?: "default" | "large";
}

withDefaults(defineProps<Props>(), {
    variant: "neutral",
    size: "default",
    label: undefined,
});

const classes = cva({
    base: "inline-flex items-center rounded-lg",
    variants: {
        variant: {
            success: "bg-semantic-success/15 text-semantic-success",
            warning: "bg-semantic-warning/15 text-semantic-warning",
            info: "bg-semantic-info/15 text-semantic-info",
            error: "bg-semantic-error/15 text-semantic-error",
            neutral: "bg-surface-indent text-text-muted",
        },
        size: {
            default: "px-2 py-0.75 text-caption-1",
            large: "text-body-3 px-2.5 py-1",
        },
    },
});
</script>

<template>
    <span :class="classes({ variant, size })">
        <slot>
            {{ label }}
        </slot>
    </span>
</template>
