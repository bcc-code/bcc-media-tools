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
    const width = Math.max(pct(marker.end) - left, 0.4);
    return { left: `${left}%`, width: `${width}%` };
}

const playheadStyle = computed(() => ({ left: `${pct(props.current)}%` }));

const hoverLeft = ref<string | null>(null);
function onTracksMove(event: MouseEvent) {
    const el = event.currentTarget as HTMLElement;
    const rect = el.getBoundingClientRect();
    const ratio = (event.clientX - rect.left) / rect.width;
    hoverLeft.value = `${Math.min(100, Math.max(0, ratio * 100))}%`;
}

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

// ---- Zoom + ruler -----------------------------------------------------------
const zoom = ref(1);
const scroll = useTemplateRef("scroll");
const { width: viewportWidth } = useElementSize(scroll);

const NICE_STEPS = [5, 10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600, 7200];
const tickStep = computed(() => {
    const contentWidth = viewportWidth.value * zoom.value;
    if (!contentWidth || !props.duration) return 600;
    const pxPerSec = contentWidth / props.duration;
    const minSecondsPerTick = 72 / pxPerSec;
    return (
        NICE_STEPS.find((s) => s >= minSecondsPerTick) ??
        NICE_STEPS[NICE_STEPS.length - 1]!
    );
});
const ticks = computed(() => {
    const step = tickStep.value;
    const out: { seconds: number; left: string; label: string }[] = [];
    for (let s = 0; s <= props.duration; s += step) {
        out.push({
            seconds: s,
            left: `${pct(s)}%`,
            label: formatMarkerTime(s),
        });
    }
    return out;
});

function keepPlayheadVisible() {
    const el = scroll.value;
    if (!el || !props.duration) return;
    const x = (props.current / props.duration) * el.scrollWidth;
    const pad = 40;
    if (x < el.scrollLeft + pad || x > el.scrollLeft + el.clientWidth - pad) {
        el.scrollTo({ left: x - el.clientWidth / 2, behavior: "smooth" });
    }
}
watch(() => props.current, keepPlayheadVisible);
watch(zoom, () => nextTick(keepPlayheadVisible));
</script>

<template>
    <div
        class="gradient-border bg-surface-default shadow-resting rounded-2xl p-3"
    >
        <div
            v-if="lanes.length === 0"
            class="text-text-hint py-6 text-center text-sm"
        >
            {{ t("markers.timeline.empty") }}
        </div>

        <template v-else>
            <div class="mb-2 flex items-center justify-end gap-2">
                <Icon name="tabler:zoom-in" class="text-text-hint size-4" />
                <DesignSlider
                    v-model="zoom"
                    :min="1"
                    :max="20"
                    :step="0.5"
                    class="w-40"
                />
            </div>

            <div class="flex gap-3">
                <div class="w-28 shrink-0">
                    <div class="mb-2 h-5" />
                    <div class="flex flex-col gap-2">
                        <div
                            v-for="lane in lanes"
                            :key="lane.value"
                            class="text-text-muted flex h-7 items-center gap-1.5 text-sm"
                        >
                            <Icon :name="lane.icon" class="size-4 shrink-0" />
                            <span class="truncate">
                                {{ laneLabel(lane.value) }}
                            </span>
                        </div>
                    </div>
                </div>

                <div ref="scroll" class="grow overflow-x-auto">
                    <div
                        class="relative"
                        :style="{ width: `${zoom * 100}%`, minWidth: '100%' }"
                    >
                        <div class="relative mb-2 h-5">
                            <div
                                v-for="tick in ticks"
                                :key="tick.seconds"
                                class="absolute inset-y-0"
                                :style="{ left: tick.left }"
                            >
                                <span
                                    class="text-text-hint absolute top-0 -translate-x-1/2 text-[10px] whitespace-nowrap tabular-nums"
                                >
                                    {{ tick.label }}
                                </span>
                                <span
                                    class="bg-border-1 absolute bottom-0 h-1.5 w-px"
                                />
                            </div>
                        </div>

                        <div
                            class="relative flex flex-col gap-2"
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
                                    :content="
                                        marker.label || laneLabel(marker.type)
                                    "
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
                                        <span class="min-w-0 truncate px-1.5">
                                            {{
                                                marker.label ||
                                                laneLabel(marker.type)
                                            }}
                                        </span>
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
            </div>
        </template>
    </div>
</template>
