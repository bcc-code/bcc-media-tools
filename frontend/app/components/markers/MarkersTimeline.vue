<script setup lang="ts">
import { MARKER_TYPES, markerTypeMeta } from "~/utils/markers";
import type { Marker, MarkerType } from "~/utils/markers";

const props = defineProps<{
    markers: Marker[];
    duration: number;
    current: number;
    selectedId?: string;
}>();

const emit = defineEmits<{
    select: [id: string];
    seek: [seconds: number];
}>();

const { t } = useI18n();

// One lane per marker type that actually has markers, in the canonical order.
const lanes = computed(() =>
    MARKER_TYPES.filter((meta) =>
        props.markers.some((m) => m.type === meta.value),
    ).map((meta) => ({
        ...meta,
        markers: props.markers.filter((m) => m.type === meta.value),
    })),
);

function pct(seconds: number) {
    if (!props.duration) return 0;
    return Math.min(100, Math.max(0, (seconds / props.duration) * 100));
}

function blockStyle(marker: Marker) {
    const left = pct(marker.start);
    const width = Math.max(pct(marker.end) - left, 0.6);
    return { left: `${left}%`, width: `${width}%` };
}

const playheadStyle = computed(() => ({ left: `${pct(props.current)}%` }));

// Click anywhere on a lane background to scrub there.
function onLaneClick(event: MouseEvent) {
    if (!props.duration) return;
    const el = event.currentTarget as HTMLElement;
    const rect = el.getBoundingClientRect();
    const ratio = (event.clientX - rect.left) / rect.width;
    emit("seek", Math.min(props.duration, Math.max(0, ratio * props.duration)));
}

function laneLabel(type: MarkerType) {
    return t(`markers.types.${type}`);
}
</script>

<template>
    <div class="border-border-1 bg-surface-default rounded-xl border p-3">
        <div
            v-if="lanes.length === 0"
            class="text-text-hint py-6 text-center text-sm"
        >
            {{ t("markers.timeline.empty") }}
        </div>

        <div v-else class="flex flex-col gap-2">
            <div
                v-for="lane in lanes"
                :key="lane.value"
                class="flex items-center gap-3"
            >
                <div
                    class="text-text-muted flex w-32 shrink-0 items-center gap-1.5 text-sm"
                >
                    <Icon :name="lane.icon" class="size-4 shrink-0" />
                    <span class="truncate">{{ laneLabel(lane.value) }}</span>
                </div>
                <div
                    class="bg-surface-indent relative h-7 grow cursor-pointer overflow-hidden rounded-md"
                    @click="onLaneClick"
                >
                    <!-- Per-lane playhead segment; stacked lanes read as one line. -->
                    <div
                        class="bg-text-default pointer-events-none absolute inset-y-0 z-20 w-px"
                        :style="playheadStyle"
                    />
                    <DesignTooltip
                        v-for="marker in lane.markers"
                        :key="marker.id"
                        :content="`${marker.label || laneLabel(marker.type)} · ${formatTime(marker.start)}–${formatTime(marker.end)}`"
                    >
                        <button
                            type="button"
                            class="ds-focus-ring absolute top-0.5 bottom-0.5 flex items-center overflow-hidden rounded px-1.5 text-left text-xs text-white/95 transition-[outline]"
                            :class="[
                                markerTypeMeta(marker.type).color,
                                selectedId === marker.id
                                    ? 'outline-text-default z-10 outline outline-2 outline-offset-1'
                                    : 'opacity-80 hover:opacity-100',
                            ]"
                            :style="blockStyle(marker)"
                            @click.stop="
                                emit('select', marker.id);
                                emit('seek', marker.start);
                            "
                        >
                            <span class="truncate">{{
                                marker.label || laneLabel(marker.type)
                            }}</span>
                        </button>
                    </DesignTooltip>
                </div>
            </div>
        </div>
    </div>
</template>
