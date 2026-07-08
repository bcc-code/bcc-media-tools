<script setup lang="ts">
const props = defineProps<{
    min: number;
    max: number;
    current: number;
    zoom: number;
}>();

const start = defineModel("start", { default: 0 });
const end = defineModel("end", { default: 0 });

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

type Drag = "move" | "start" | "end";
const dragging = ref<Drag>();
const setDragging = (d: Drag) => {
    if (dragging.value != undefined) return;
    dragging.value = d;
};

function onDrag(event: Event) {
    if (!(event instanceof MouseEvent)) return;
    if (dragging.value === undefined) return;

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
        class="border-text-default h-32 w-full overflow-hidden border"
    >
        <div
            class="relative h-full"
            :style="{ width: `${props.max * props.zoom}px` }"
        >
            <div
                class="bg-surface-default absolute h-full cursor-grab active:cursor-grabbing"
                :style
                @mousedown.stop="setDragging('move')"
            >
                <div
                    class="bg-text-default absolute left-0 h-full w-1 cursor-w-resize"
                    @mousedown.stop="setDragging('start')"
                />
                <div
                    class="bg-text-default absolute right-0 h-full w-1 cursor-e-resize"
                    @mousedown.stop="setDragging('end')"
                />
            </div>
            <div
                class="pointer-events-none absolute z-10 h-full border-r border-dotted"
                :style="{ left: currentLeft }"
            />
        </div>
    </div>
</template>
