<script setup lang="ts">
import { formatMarkerTime, markerTypeMeta } from "~/utils/markers";
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
        class="ds-focus-ring flex w-full items-center gap-3 rounded-lg px-3 py-1.5 text-left transition-colors"
        :class="selected ? 'bg-surface-indent' : 'hover:bg-surface-indent/60'"
        @click="$emit('select')"
    >
        <Icon
            :name="markerTypeMeta(marker.type).icon"
            class="size-4 shrink-0"
            :class="markerTypeMeta(marker.type).iconColor"
        />
        <span class="text-text-hint text-caption-1 w-40 shrink-0 tabular-nums">
            {{ formatMarkerTime(marker.start) }} –
            {{ formatMarkerTime(marker.end) }}
        </span>
        <span class="text-body-3 text-text-default truncate">
            {{ marker.label || t(`markers.types.${marker.type}`) }}
        </span>
        <Icon
            v-if="active"
            name="tabler:volume"
            class="text-primary-default size-3.5 shrink-0"
        />
        <DesignBadge
            v-if="marker.source === 'imported'"
            variant="info"
            class="ml-auto shrink-0"
        >
            {{ t("markers.source.imported") }}
        </DesignBadge>
    </button>
</template>
