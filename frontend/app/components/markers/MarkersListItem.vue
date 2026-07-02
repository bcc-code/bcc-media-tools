<script setup lang="ts">
import {
    formatMarkerDuration,
    isMarkerUnresolved,
    markerTypeMeta,
} from "~/utils/markers";
import type { Marker } from "~/utils/markers";

const props = defineProps<{
    marker: Marker;
    selected: boolean;
    currentTime: number;
}>();

defineEmits<{ select: [] }>();

const { t } = useI18n();

const active = computed(
    () =>
        props.currentTime >= props.marker.start &&
        props.currentTime <= props.marker.end,
);

// Bring the row into view when it becomes the selected one (e.g. selected from
// the timeline or after adding), without scrolling if it's already visible.
const el = useTemplateRef("el");
watch(
    () => props.selected,
    (isSelected) => {
        if (isSelected)
            nextTick(() =>
                el.value?.scrollIntoView({
                    block: "nearest",
                    behavior: "smooth",
                }),
            );
    },
);

const displayLabel = computed(
    () => props.marker.label || t(`markers.types.${props.marker.type}`),
);

const unresolved = computed(() => isMarkerUnresolved(props.marker));
</script>

<template>
    <button
        ref="el"
        type="button"
        :title="displayLabel"
        class="ds-focus-ring flex w-full items-center gap-3 rounded-lg px-3 py-1.5 text-left transition-colors"
        :class="
            selected
                ? 'bg-surface-indent'
                : active
                  ? 'bg-surface-indent/40'
                  : 'hover:bg-surface-indent/60'
        "
        @click="$emit('select')"
    >
        <Icon
            :name="markerTypeMeta(marker.type).icon"
            class="size-4 shrink-0"
            :class="markerTypeMeta(marker.type).iconColor"
        />
        <span class="text-body-3 text-text-default flex-1 truncate">
            {{ displayLabel }}
        </span>
        <DesignTooltip
            v-if="unresolved"
            :content="t('markers.review.hint')"
            placement="top"
        >
            <Icon
                name="tabler:alert-triangle"
                class="text-semantic-warning size-3.5 shrink-0"
            />
        </DesignTooltip>
        <Icon
            v-if="active"
            name="tabler:volume"
            class="text-primary-default size-3.5 shrink-0"
        />
        <span class="text-text-hint text-caption-1 shrink-0 tabular-nums">
            {{ formatMarkerDuration(marker.end - marker.start) }}
        </span>
    </button>
</template>
