<script setup lang="ts">
import {
    MARKER_TYPES,
    formatMarkerTime,
    markerTypeMeta,
} from "~/utils/markers";
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

// Muted hover playhead — previews where a click on the tracks would seek.
const hoverLeft = ref<string | null>(null);
function onTracksMove(event: MouseEvent) {
    const el = event.currentTarget as HTMLElement;
    const rect = el.getBoundingClientRect();
    const ratio = (event.clientX - rect.left) / rect.width;
    hoverLeft.value = `${Math.min(100, Math.max(0, ratio * 100))}%`;
}

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

        <div v-else class="flex gap-3">
            <div class="flex w-32 shrink-0 flex-col gap-2">
                <div
                    v-for="lane in lanes"
                    :key="lane.value"
                    class="text-text-muted flex h-7 items-center gap-1.5 text-sm"
                >
                    <Icon :name="lane.icon" class="size-4 shrink-0" />
                    <span class="truncate">{{ laneLabel(lane.value) }}</span>
                </div>
            </div>

            <div
                class="relative flex grow flex-col gap-2"
                @mousemove="onTracksMove"
                @mouseleave="hoverLeft = null"
            >
                <div
                    v-for="lane in lanes"
                    :key="lane.value"
                    class="bg-surface-indent relative h-7 cursor-pointer overflow-hidden rounded-md"
                    @click="onLaneClick"
                >
                    <DesignTooltip
                        v-for="marker in lane.markers"
                        :key="marker.id"
                        :content="`${marker.label || laneLabel(marker.type)} · ${formatMarkerTime(marker.start)}–${formatMarkerTime(marker.end)}`"
                    >
                        <button
                            type="button"
                            class="ds-focus-ring absolute top-0.5 bottom-0.5 flex items-center overflow-hidden rounded text-left text-xs text-white/95 transition-[outline]"
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
                            <span class="min-w-0 truncate px-1.5">{{
                                marker.label || laneLabel(marker.type)
                            }}</span>
                        </button>
                    </DesignTooltip>
                </div>

                <div
                    v-if="hoverLeft"
                    class="bg-text-default/30 pointer-events-none absolute inset-y-0 z-10 w-px"
                    :style="{ left: hoverLeft }"
                />
                <div
                    class="bg-text-default pointer-events-none absolute inset-y-0 z-20 w-px"
                    :style="playheadStyle"
                />
            </div>
        </div>
    </div>
</template>
