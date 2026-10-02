<script lang="ts" setup>
import type { EditorialRow } from "~/utils/editorial";
import { editorialTypeColor, formatMs, parseTc } from "~/utils/editorial";

const props = defineProps<{
    rows: EditorialRow[];
    /** Index of the item shown in the detail panel. */
    selectedIndex: number;
    /** Playhead position in milliseconds. */
    currentMs: number;
    /** Length of the whole programme in milliseconds. */
    totalMs: number;
}>();

const emit = defineEmits<{
    select: [number];
    seek: [number];
}>();

const { typeLabel } = useEditorialTypes();

// A programme runs anything from a 40-second prayer to a half-hour speech, so
// strictly proportional blocks leave the short items a few pixels wide — and
// clamping those to a readable width flattens everything else to the same size.
// Widths follow the square root of the duration instead: long items still read
// as long (a 14-minute speech is ~3x a 1-minute prayer rather than 14x), short
// ones stay legible, and MIN_PX only has to catch the extremes. What that costs
// is taken from the blocks that have room to give.
const MIN_PX = 48;
const GAP_PX = 4;

const wrapperEl = useTemplateRef<HTMLElement>("wrapperEl");
// Measured on the scroll container, not the laid-out track, so the measurement
// never feeds back into the width it produces.
const { width: wrapperWidth } = useElementSize(wrapperEl);

interface Block {
    row: EditorialRow;
    index: number;
    startMs: number;
    endMs: number;
    left: number;
    width: number;
    color: string;
}

const blocks = computed<Block[]>(() => {
    const rows = props.rows;
    const n = rows.length;
    if (n === 0) return [];

    const spans = rows.map((row) => {
        const start = parseTc(row.start);
        return { start, end: Math.max(start + 1, parseTc(row.end)) };
    });
    const weights = spans.map((s) => Math.sqrt(s.end - s.start));
    const totalWeight = weights.reduce((sum, w) => sum + w, 0);

    const gaps = GAP_PX * (n - 1);
    // When even the minimums don't fit, the track grows and the wrapper scrolls.
    const available = Math.max(
        n * MIN_PX,
        (wrapperWidth.value || n * MIN_PX + gaps) - gaps,
    );

    let widths = weights.map((w) => (w / totalWeight) * available);
    // Lift every block to the minimum, paying for it from the blocks above it.
    // A few passes settle it; each one can only push more blocks to the floor.
    for (let pass = 0; pass < 4; pass++) {
        const deficit = widths.reduce(
            (sum, w) => sum + Math.max(0, MIN_PX - w),
            0,
        );
        if (deficit < 0.5) break;
        const spare = widths.reduce(
            (sum, w) => sum + Math.max(0, w - MIN_PX),
            0,
        );
        if (spare < 0.5) break;
        widths = widths.map((w) =>
            w <= MIN_PX
                ? MIN_PX
                : Math.max(MIN_PX, w - ((w - MIN_PX) / spare) * deficit),
        );
    }

    let x = 0;
    return widths.map((width, index) => {
        const block: Block = {
            row: rows[index]!,
            index,
            startMs: spans[index]!.start,
            endMs: spans[index]!.end,
            left: x,
            width,
            color: editorialTypeColor(rows[index]!.type),
        };
        x += width + GAP_PX;
        return block;
    });
});

const contentWidth = computed(() => {
    const last = blocks.value[blocks.value.length - 1];
    return last ? last.left + last.width : 0;
});

// Blocks no longer sit where raw time would put them, so time ↔ pixel runs
// through the layout: inside a block it interpolates, between blocks it sticks
// to the nearest edge. The ruler and the playhead use it, so what they point at
// is always the block underneath.
function timeToX(ms: number): number {
    const list = blocks.value;
    if (!list.length) return 0;
    const first = list[0]!;
    if (ms <= first.startMs) return first.left;
    for (const b of list) {
        if (ms < b.startMs) return b.left;
        if (ms <= b.endMs) {
            return (
                b.left + ((ms - b.startMs) / (b.endMs - b.startMs)) * b.width
            );
        }
    }
    const last = list[list.length - 1]!;
    return last.left + last.width;
}

function xToTime(x: number): number {
    const list = blocks.value;
    if (!list.length) return 0;
    for (const b of list) {
        if (x < b.left) return b.startMs;
        if (x <= b.left + b.width) {
            return b.startMs + ((x - b.left) / b.width) * (b.endMs - b.startMs);
        }
    }
    return list[list.length - 1]!.endMs;
}

const playheadX = computed(() => timeToX(props.currentMs));

// Ruler ticks: the smallest step from the ladder that keeps them sparse, then
// anything that would collide with its neighbour after the squeeze is dropped.
const TICK_STEPS_MS = [
    30_000,
    60_000,
    2 * 60_000,
    5 * 60_000,
    10 * 60_000,
    15 * 60_000,
    30 * 60_000,
    60 * 60_000,
];
const MIN_TICK_GAP_PX = 64;

const ticks = computed(() => {
    const total = Math.max(1, props.totalMs);
    const step =
        TICK_STEPS_MS.find((s) => total / s <= 10) ??
        TICK_STEPS_MS[TICK_STEPS_MS.length - 1]!;

    const out: { ms: number; x: number; label: string }[] = [];
    let lastX = -Infinity;
    for (let ms = 0; ms < total; ms += step) {
        const x = timeToX(ms);
        if (x - lastX < MIN_TICK_GAP_PX) continue;
        lastX = x;
        out.push({ ms, x, label: formatMs(ms).replace(/^00:/, "") });
    }
    return out;
});

// The scrub lane seeks anywhere in the programme; the blocks below jump to the
// start of an item, which is what reviewing one actually calls for.
const laneEl = useTemplateRef<HTMLElement>("laneEl");

function seekFromPointer(e: PointerEvent) {
    const el = laneEl.value;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    emit("seek", xToTime(e.clientX - rect.left));
}

function onLanePointerDown(e: PointerEvent) {
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    seekFromPointer(e);
}

function onLanePointerMove(e: PointerEvent) {
    if (e.buttons !== 1) return;
    seekFromPointer(e);
}
</script>

<template>
    <div ref="wrapperEl" class="scrollbar-hide overflow-x-auto">
        <div
            class="relative select-none"
            :style="{ width: `${Math.max(contentWidth, 1)}px` }"
        >
            <!-- Scrub lane: drag anywhere to move the playhead. -->
            <div
                ref="laneEl"
                class="bg-text-default/10 dark:bg-surface-indent relative h-6 cursor-pointer rounded-t-lg"
                @pointerdown="onLanePointerDown"
                @pointermove="onLanePointerMove"
            >
                <div
                    class="bg-primary-default/30 pointer-events-none absolute inset-y-0 left-0 rounded-tl-lg"
                    :style="{ width: `${playheadX}px` }"
                />
                <div
                    class="bg-primary-default pointer-events-none absolute inset-y-0 -ml-px w-0.5"
                    :style="{ left: `${playheadX}px` }"
                />
            </div>

            <!-- Item blocks. The name is deliberately not here: it fits on
                 two blocks out of twenty-five, and the detail panel above is
                 already showing it for the selected item. What the strip is for
                 is order, type, length, publish state and jumping between
                 items — so it carries the number, the duration and the two
                 publish bars, and the title lives in the tooltip. -->
            <div class="relative h-20">
                <DesignTooltip
                    v-for="block in blocks"
                    :key="block.row.id || `new-${block.index}`"
                    placement="top"
                    :open-delay="150"
                >
                    <button
                        type="button"
                        class="ds-focus-ring absolute inset-y-0 flex flex-col overflow-hidden rounded-lg px-2 py-1.5 text-left transition-[opacity,box-shadow] duration-200"
                        :class="
                            block.index === selectedIndex
                                ? 'ring-text-default z-10 opacity-100 ring-2'
                                : 'opacity-85 hover:opacity-100'
                        "
                        :style="{
                            left: `${block.left}px`,
                            width: `${block.width}px`,
                            backgroundColor: block.color,
                        }"
                        @click="emit('select', block.index)"
                    >
                        <span
                            class="text-title-2 text-text-dark-default tabular-nums"
                        >
                            {{ String(block.index + 1).padStart(2, "0") }}
                        </span>
                        <span
                            v-if="block.width > 76"
                            class="text-caption-1 text-text-dark-muted tabular-nums"
                        >
                            {{ formatMs(block.endMs - block.startMs) }}
                        </span>
                        <!-- Publish state: left bar is BMM, right is BCC Media. -->
                        <span class="mt-auto flex gap-1.5">
                            <span
                                class="h-1.5 w-4 rounded-full"
                                :class="
                                    block.row.publishBmm
                                        ? 'bg-text-dark-default'
                                        : 'bg-text-dark-default/20'
                                "
                            />
                            <span
                                class="h-1.5 w-4 rounded-full"
                                :class="
                                    block.row.publishBcc
                                        ? 'bg-text-dark-default'
                                        : 'bg-text-dark-default/20'
                                "
                            />
                        </span>
                    </button>

                    <template #content>
                        <p class="text-title-3 text-text-default">
                            {{ block.row.name || "—" }}
                        </p>
                        <p
                            v-if="block.row.contributors"
                            class="text-text-default mt-0.5"
                        >
                            {{ block.row.contributors }}
                        </p>
                        <p class="text-text-muted mt-0.5">
                            {{ typeLabel(block.row.type) }} ·
                            {{ formatMs(block.endMs - block.startMs) }}
                        </p>
                    </template>
                </DesignTooltip>

                <div
                    class="bg-primary-default pointer-events-none absolute inset-y-0 z-20 -ml-px w-0.5"
                    :style="{ left: `${playheadX}px` }"
                />
            </div>

            <!-- Time ruler -->
            <div class="relative h-6">
                <div
                    v-for="tick in ticks"
                    :key="tick.ms"
                    class="border-border-1 absolute top-0 h-full border-l pl-1"
                    :style="{ left: `${tick.x}px` }"
                >
                    <span class="text-body-3 text-text-muted tabular-nums">
                        {{ tick.label }}
                    </span>
                </div>
            </div>
        </div>
    </div>
</template>
