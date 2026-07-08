<script setup lang="ts">
const props = defineProps<{
    min: number;
    max: number;
    current: number;
    zoom: number;
    vxid: string;
    base: string;
}>();

const start = defineModel("start", { default: 0 });
const end = defineModel("end", { default: 0 });

const emit = defineEmits<{ seek: [time: number] }>();

// Filmstrip. Two independent concerns:
//  - We only ever fetch from a fixed pool of fractions (GRID) so the backend
//    LRU cache stays warm and zooming never triggers a re-fetch storm.
//  - How many frames we *show* follows the track width, so each thumbnail keeps
//    a natural ~16:9 aspect instead of being squished into a sliver. When there
//    are fewer slots than pool entries we subsample the pool; the visible
//    frames always map to already-cached fractions.
const TRACK_HEIGHT = 128; // matches h-32 on the track
const TARGET_THUMB_WIDTH = Math.round((TRACK_HEIGHT * 16) / 9); // ~16:9

const GRID = computed(() =>
    Math.min(80, Math.max(8, Math.round(props.max / 4))),
);

const frames = computed(() => {
    if (!props.vxid || !props.base || props.max <= 0) return [];
    const grid = GRID.value;
    const trackWidth = props.max * props.zoom;
    const count = Math.min(
        grid,
        Math.max(1, Math.round(trackWidth / TARGET_THUMB_WIDTH)),
    );
    return Array.from({ length: count }, (_, i) => {
        const gridIndex =
            count === 1 ? 0 : Math.round((i * (grid - 1)) / (count - 1));
        return {
            f: (gridIndex + 0.5) / grid,
            left: `${(i / count) * 100}%`,
            width: `${(1 / count) * 100}%`,
        };
    });
});
const thumbUrl = (f: number) =>
    `${props.base}/vault/thumbnail?vxid=${encodeURIComponent(props.vxid)}&f=${f}`;

// Percentage bounds of the selected span, used to dim everything outside it.
const startPct = computed(() => `${(start.value / props.max) * 100}%`);
const endPct = computed(() => `${(end.value / props.max) * 100}%`);

// Timecode ruler. One second is `zoom` pixels wide, so pick the smallest "nice"
// interval whose spacing is at least ~70px to keep labels from colliding.
const NICE_INTERVALS = [
    1, 2, 5, 10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600,
];
const tickInterval = computed(() => {
    for (const n of NICE_INTERVALS) {
        if (n * props.zoom >= 70) return n;
    }
    return NICE_INTERVALS[NICE_INTERVALS.length - 1]!;
});
const ticks = computed(() => {
    const iv = tickInterval.value;
    const out: { t: number; left: string }[] = [];
    for (let t = 0; t <= props.max; t += iv) {
        out.push({ t, left: `${(t / props.max) * 100}%` });
    }
    return out;
});
const tickLabel = (t: number) => {
    const m = Math.floor(t / 60);
    const s = Math.floor(t % 60);
    return `${m}:${String(s).padStart(2, "0")}`;
};

onMounted(() => {
    start.value = props.min;
    end.value = props.max;
});

const style = computed(() => {
    return {
        width: `${((end.value - start.value) / props.max) * 100}%`,
        left: `${(start.value / props.max) * 100}%`,
    };
});

const currentLeft = computed(() => {
    return `${(props.current / props.max) * 100}%`;
});

const scroller = useTemplateRef("scroller");
watch([() => props.current, () => props.zoom], () => {
    const el = scroller.value;
    if (!el) return;
    const playheadX = props.current * props.zoom;
    // Only recenter when the playhead has left the visible window, so normal
    // playback doesn't trigger a continuous smooth-scroll.
    if (
        playheadX < el.scrollLeft ||
        playheadX > el.scrollLeft + el.clientWidth
    ) {
        el.scrollTo({
            left: playheadX - el.clientWidth / 2,
            behavior: "smooth",
        });
    }
});

type Drag = "move" | "start" | "end" | "seek";
const dragging = ref<Drag>();
const setDragging = (d: Drag) => {
    if (dragging.value != undefined) return;
    dragging.value = d;
};

const track = useTemplateRef("track");
// Seek the playhead to the clicked position. Click/drag anywhere on the track
// background (filmstrip gaps + ruler) scrubs; handles and the selection body
// stop propagation, so trimming/moving is unaffected.
function seekFromEvent(event: MouseEvent) {
    const el = track.value;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const time = (event.clientX - rect.left) / props.zoom;
    emit("seek", Math.min(Math.max(time, props.min), props.max));
}
function startSeek(event: MouseEvent) {
    setDragging("seek");
    seekFromEvent(event);
}

function onDrag(event: Event) {
    if (!(event instanceof MouseEvent)) return;
    if (dragging.value === undefined) return;

    if (dragging.value == "seek") {
        seekFromEvent(event);
        return;
    }

    const delta = event.movementX / props.zoom;

    if (dragging.value == "move") {
        start.value = start.value + delta;
        end.value = end.value + delta;
    }

    if (dragging.value == "start") {
        // Don't let the start handle move past the end handle.
        start.value = Math.min(start.value + delta, end.value);
    }

    if (dragging.value == "end") {
        // Don't let the end handle move before the start handle.
        end.value = Math.max(end.value + delta, start.value);
    }
}

useEventListener("mousemove", onDrag);
useEventListener("mouseup", () => {
    dragging.value = undefined;
});

watch([start, end], ([s, e]) => {
    if (s < props.min) start.value = props.min;
    if (e > props.max) end.value = props.max;
});
</script>

<template>
    <div
        ref="scroller"
        class="border-text-default w-full overflow-hidden border select-none"
    >
        <div
            ref="track"
            class="relative cursor-pointer"
            :style="{ width: `${props.max * props.zoom}px` }"
            @mousedown="startSeek"
        >
            <div class="bg-surface-indent relative h-32 w-full">
                <img
                    v-for="frame in frames"
                    :key="frame.f"
                    :src="thumbUrl(frame.f)"
                    class="pointer-events-none absolute top-0 h-full border-r border-black/30 object-cover"
                    :style="{ left: frame.left, width: frame.width }"
                    loading="lazy"
                    draggable="false"
                    alt=""
                    @error="
                        (e) =>
                            ((e.target as HTMLImageElement).style.visibility =
                                'hidden')
                    "
                />
                <div
                    class="pointer-events-none absolute top-0 left-0 z-1 h-full bg-black/60"
                    :style="{ width: startPct }"
                />
                <div
                    class="pointer-events-none absolute top-0 z-1 h-full bg-black/60"
                    :style="{ left: endPct, right: 0 }"
                />
                <div
                    class="absolute z-2 h-full cursor-grab shadow-[inset_0_0_0_2px_rgba(255,255,255,0.95),inset_0_0_0_3px_rgba(0,0,0,0.55)] active:cursor-grabbing"
                    :style
                    @mousedown.stop="setDragging('move')"
                >
                    <div
                        class="absolute left-0 h-full w-2 cursor-w-resize bg-white shadow-[0_0_0_1.5px_rgba(0,0,0,0.7)]"
                        @mousedown.stop="setDragging('start')"
                    >
                        <div
                            class="absolute top-1/2 left-1/2 h-6 w-px -translate-x-1/2 -translate-y-1/2 bg-black/60"
                        />
                    </div>
                    <div
                        class="absolute right-0 h-full w-2 cursor-e-resize bg-white shadow-[0_0_0_1.5px_rgba(0,0,0,0.7)]"
                        @mousedown.stop="setDragging('end')"
                    >
                        <div
                            class="absolute top-1/2 left-1/2 h-6 w-px -translate-x-1/2 -translate-y-1/2 bg-black/60"
                        />
                    </div>
                </div>
            </div>
            <div class="bg-surface-default text-text-hint relative h-6 w-full">
                <div
                    v-for="tick in ticks"
                    :key="tick.t"
                    class="pointer-events-none absolute top-0 flex flex-col items-start"
                    :style="{ left: tick.left }"
                >
                    <div class="bg-text-muted h-1.5 w-px" />
                    <span class="pl-1 text-[10px] leading-none tabular-nums">
                        {{ tickLabel(tick.t) }}
                    </span>
                </div>
            </div>
            <div
                class="pointer-events-none absolute top-0 z-20 h-full w-0.5 bg-white shadow-[0_0_0_1px_rgba(0,0,0,0.7)]"
                :style="{ left: currentLeft }"
            />
        </div>
    </div>
</template>
