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

const currentElement = useTemplateRef("currentElement");
watch([currentLeft, () => props.zoom], () => {
    if (currentElement.value) {
        currentElement.value.scrollIntoView({
            inline: "center",
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
        start.value = start.value + delta;
    }

    if (dragging.value == "end") {
        end.value = end.value + delta;
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
    <div class="border-inverted h-32 w-full overflow-hidden border">
        <div
            class="relative h-full"
            :style="{ width: `${props.max * props.zoom}px` }"
        >
            <div
                class="bg-default text-inverted absolute h-full cursor-grab active:cursor-grabbing"
                :style
                @mousedown.stop="setDragging('move')"
            >
                <div
                    class="bg-inverted absolute left-0 h-full w-1 cursor-w-resize"
                    @mousedown.stop="setDragging('start')"
                />
                <div
                    class="bg-inverted absolute right-0 h-full w-1 cursor-e-resize"
                    @mousedown.stop="setDragging('end')"
                />
            </div>
            <div
                ref="currentElement"
                class="pointer-events-none absolute z-10 h-full border-r border-dotted"
                :style="{ left: currentLeft }"
            />
        </div>
    </div>
</template>
