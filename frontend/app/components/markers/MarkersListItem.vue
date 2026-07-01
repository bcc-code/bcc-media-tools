<script setup lang="ts">
import { markerTypeMeta } from "~/utils/markers";
import type { Marker } from "~/utils/markers";

const props = defineProps<{
    marker: Marker;
    selected: boolean;
    currentTime: number;
}>();

defineEmits<{ select: [] }>();

const { t } = useI18n();

// Highlight markers whose in/out range contains the playhead.
const active = computed(
    () =>
        props.currentTime >= props.marker.start &&
        props.currentTime <= props.marker.end,
);
</script>

<template>
    <button
        type="button"
        class="ds-focus-ring flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors"
        :class="selected ? 'bg-surface-indent' : 'hover:bg-surface-indent/60'"
        @click="$emit('select')"
    >
        <span
            class="size-2.5 shrink-0 rounded-full"
            :class="markerTypeMeta(marker.type).color"
        />
        <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
                <span class="text-body-3 text-text-default truncate">
                    {{ marker.label || t(`markers.types.${marker.type}`) }}
                </span>
                <Icon
                    v-if="active"
                    name="tabler:volume"
                    class="text-primary-default size-3.5 shrink-0"
                />
            </div>
            <span class="text-text-hint text-caption-1 tabular-nums">
                {{ formatTime(marker.start) }} – {{ formatTime(marker.end) }}
            </span>
        </div>
        <DesignBadge v-if="marker.source === 'imported'" variant="info">
            {{ t("markers.source.imported") }}
        </DesignBadge>
    </button>
</template>
