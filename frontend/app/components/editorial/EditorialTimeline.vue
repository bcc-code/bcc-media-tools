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

interface Block {
    row: EditorialRow;
    index: number;
    left: number;
    width: number;
    color: string;
}

const blocks = computed<Block[]>(() => {
    const total = Math.max(1, props.totalMs);
    return props.rows.map((row, index) => {
        const start = parseTc(row.start);
        const end = Math.max(start, parseTc(row.end));
        return {
            row,
            index,
            left: (start / total) * 100,
            width: Math.max((end - start) / total, 0) * 100,
            color: editorialTypeColor(row.type),
        };
    });
});

const playheadPercent = computed(
    () => (props.currentMs / Math.max(1, props.totalMs)) * 100,
);

// Ruler ticks: pick the smallest step from the ladder that keeps the labels
// from colliding (aiming for at most ~10 of them).
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
const ticks = computed(() => {
    const total = Math.max(1, props.totalMs);
    const step =
        TICK_STEPS_MS.find((s) => total / s <= 10) ??
        TICK_STEPS_MS[TICK_STEPS_MS.length - 1]!;
    const out: { ms: number; left: number; label: string }[] = [];
    for (let ms = 0; ms < total; ms += step) {
        out.push({
            ms,
            left: (ms / total) * 100,
            label: formatMs(ms).replace(/^00:/, ""),
        });
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
    const ratio = Math.min(
        1,
        Math.max(0, (e.clientX - rect.left) / rect.width),
    );
    emit("seek", ratio * props.totalMs);
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
    <div class="scrollbar-hide overflow-x-auto">
        <div class="min-w-[56rem] select-none">
            <!-- Scrub lane: drag anywhere to move the playhead. -->
            <div
                ref="laneEl"
                class="bg-text-default/10 dark:bg-surface-indent relative h-6 cursor-pointer rounded-t-lg"
                @pointerdown="onLanePointerDown"
                @pointermove="onLanePointerMove"
            >
                <div
                    class="bg-primary-default/30 pointer-events-none absolute inset-y-0 left-0 rounded-tl-lg"
                    :style="{ width: `${playheadPercent}%` }"
                />
                <div
                    class="bg-primary-default pointer-events-none absolute inset-y-0 -ml-px w-0.5"
                    :style="{ left: `${playheadPercent}%` }"
                />
            </div>

            <!-- Item blocks, laid out proportionally over the programme length. -->
            <div class="relative h-28">
                <button
                    v-for="block in blocks"
                    :key="block.row.id || `new-${block.index}`"
                    type="button"
                    class="ds-focus-ring absolute inset-y-0 overflow-hidden rounded-lg px-2 py-1.5 text-left transition-[opacity,box-shadow] duration-200"
                    :class="
                        block.index === selectedIndex
                            ? 'ring-text-default z-10 opacity-100 ring-2'
                            : 'opacity-65 hover:opacity-90'
                    "
                    :style="{
                        left: `${block.left}%`,
                        width: `max(0.75rem, ${block.width}%)`,
                        backgroundColor: block.color,
                    }"
                    :title="`${block.row.name || '—'} · ${typeLabel(block.row.type)}`"
                    @click="emit('select', block.index)"
                >
                    <span
                        class="text-title-1 text-text-dark-default block tabular-nums"
                    >
                        {{ String(block.index + 1).padStart(2, "0") }}
                    </span>
                    <span
                        v-if="block.width > 5"
                        class="text-body-3 text-text-dark-muted mt-1.5 line-clamp-2 block"
                    >
                        {{ block.row.name }}
                    </span>
                    <!-- Publish state: left bar is BMM, right bar is BCC Media. -->
                    <span class="absolute bottom-2 left-2 flex gap-1.5">
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
            </div>

            <!-- Time ruler -->
            <div class="relative h-6">
                <div
                    v-for="tick in ticks"
                    :key="tick.ms"
                    class="border-border-1 absolute top-0 h-full border-l pl-1"
                    :style="{ left: `${tick.left}%` }"
                >
                    <span class="text-body-3 text-text-muted tabular-nums">
                        {{ tick.label }}
                    </span>
                </div>
            </div>
        </div>
    </div>
</template>
