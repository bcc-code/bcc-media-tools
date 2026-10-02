<script lang="ts" setup>
import type { ComponentPublicInstance } from "vue";

const props = defineProps<{
    focusedSegment?: Segment;
    mediaDuration?: number;
}>();

const emit = defineEmits<{
    wordFocus: [Word, Segment];
    /** Fired just before a destructive change, so it can be undone as one step. */
    snapshot: [];
}>();

const segments = defineModel<Segment[]>({ required: true });

const handleSegmentUpdate = (uid: string, segment: Segment) => {
    segments.value = updateSegment(segments.value, uid, () => segment);
};

// Marks the row rather than removing it, so the list the user sees and the one
// we submit stay the same shape. `toTranscription()` drops marked rows.
const handleSegmentToggleDelete = (uid: string) => {
    emit("snapshot");
    segments.value = toggleSegmentDeleted(segments.value, uid);
};

const handleAddSegment = (index: number) => {
    segments.value = insertSegmentAt(
        segments.value,
        index,
        props.mediaDuration,
    );
};

const { deleteMode } = useDeleteMode();

/*
 * Marking rows in delete mode.
 *
 * Clicking row by row is fine for a stray line, but removing an intro, a song or
 * a tail of dead air means tens or hundreds of rows. Two gestures cover that:
 * press and drag to paint a run, or click one row and shift-click another to
 * mark everything between them. Both work the same way — the row pressed first
 * is the anchor, and the range from it to the row under the pointer takes the
 * anchor's new state.
 *
 * Each gesture is applied to the document as it was when the gesture began
 * (`gestureBase`), never to the result of the previous pointer move. That is
 * what lets a drag be taken back by dragging the other way: rows that fall out
 * of the range return to what they were, instead of keeping whatever the drag
 * painted on the way past.
 */
const anchor = ref<string | null>(null);
const painting = ref<boolean | null>(null);
let gestureBase: Segment[] | null = null;

/*
 * What the gesture would touch, shown before it is committed.
 *
 * Holding shift is a promise about rows that have not changed yet, so the range
 * from the anchor to the row under the pointer is highlighted while the key is
 * down. A drag applies as it goes, but the same highlight marks out which rows
 * belong to *this* gesture rather than to one made earlier.
 */
const hovered = ref<string | null>(null);
const shiftHeld = ref(false);

const preview = computed(() =>
    deleteMode.value
        ? previewRange(segments.value, {
              anchorUid: anchor.value,
              hoveredUid: hovered.value,
              painting: painting.value,
              shiftHeld: shiftHeld.value,
          })
        : null,
);

// The row is resolved by position rather than from `event.target`, because a
// touch drag keeps sending its events to the row it started on — and because
// the pointer can end up over a different row without moving at all, when the
// list is scrolled under it.
const uidAtPoint = (x: number, y: number): string | null => {
    const el = document.elementFromPoint(x, y);
    return (
        el?.closest<HTMLElement>("[data-segment-uid]")?.dataset.segmentUid ??
        null
    );
};

let lastPoint: { x: number; y: number } | null = null;

const applyRange = (base: Segment[], toUid: string, deleted: boolean) => {
    if (!anchor.value) return;
    const next = setSegmentsDeleted(
        base,
        segmentRange(base, anchor.value, toUid),
        deleted,
    );
    if (next !== segments.value) segments.value = next;
};

const handlePointerDown = (event: PointerEvent) => {
    if (!deleteMode.value || event.button !== 0) return;

    lastPoint = { x: event.clientX, y: event.clientY };
    const uid = uidAtPoint(event.clientX, event.clientY);
    if (!uid) return;
    const segment = segments.value.find((s) => s.uid === uid);
    if (!segment) return;

    // Keeps the press from starting a text selection across the rows.
    event.preventDefault();
    emit("snapshot");
    gestureBase = segments.value;

    // Shift extends from the existing anchor, and the range takes the anchor's
    // state — so shift-clicking after marking a row marks everything down to
    // there, and after unmarking one it clears everything down to there.
    if (event.shiftKey && anchor.value) {
        const anchored = segments.value.find((s) => s.uid === anchor.value);
        applyRange(gestureBase, uid, !!anchored?.deleted);
        return;
    }

    anchor.value = uid;
    painting.value = !segment.deleted;
    applyRange(gestureBase, uid, painting.value);
};

const handlePointerMove = (event: PointerEvent) => {
    lastPoint = { x: event.clientX, y: event.clientY };
    trackHovered();
    extendToPointer();
};

const trackHovered = () => {
    if (!deleteMode.value || !lastPoint) return;
    // Assigning the same uid is a no-op, so the preview is only recomputed when
    // the pointer actually crosses into another row.
    hovered.value = uidAtPoint(lastPoint.x, lastPoint.y);
};

/**
 * Also called while the list scrolls under a held pointer, so that scrolling —
 * with the wheel, or by dragging past the edge of the list — is how a range
 * longer than the window gets marked.
 */
const extendToPointer = () => {
    if (painting.value === null || !gestureBase || !lastPoint) return;

    const uid = uidAtPoint(lastPoint.x, lastPoint.y);
    if (uid) applyRange(gestureBase, uid, painting.value);
};

const handleScroll = () => {
    trackHovered();
    extendToPointer();
};

const endGesture = () => {
    painting.value = null;
    gestureBase = null;
};

onMounted(() => {
    // On window, so letting go outside the list still ends the drag.
    window.addEventListener("pointerup", endGesture);
    window.addEventListener("pointercancel", endGesture);
});

onBeforeUnmount(() => {
    window.removeEventListener("pointerup", endGesture);
    window.removeEventListener("pointercancel", endGesture);
});

// Tracked on window rather than on the rows: the key can go down and up while
// the pointer rests on a row, and the highlight has to follow it either way.
const trackShift = (event: KeyboardEvent) => {
    shiftHeld.value = event.shiftKey;
};
useEventListener(window, "keydown", trackShift);
useEventListener(window, "keyup", trackShift);
// A window that loses focus never sends the keyup.
useEventListener(window, "blur", () => (shiftHeld.value = false));

// An anchor pointing at a row that no longer exists would extend from nowhere.
watch(deleteMode, () => {
    anchor.value = null;
    hovered.value = null;
    endGesture();
});

const segmentelements = defineModel<Record<number, ComponentPublicInstance>>(
    "segmentelements",
    { default: () => ({}) },
);

function focusSegment(index: number, direction: number) {
    const next = segmentelements.value?.[index + direction];
    if (!next) return;
    const child = next.$el.querySelector(
        "[contenteditable]",
    ) as HTMLElement | null;
    child?.focus();
}

const { list, wrapperProps, containerProps } = useVirtualList<Segment>(
    computed(() => segments.value),
    { itemHeight: 80, overscan: 10 },
);

const addButtonClass =
    "bg-surface-raise border-border-1 absolute left-1/2 z-10 grid aspect-square size-6 -translate-x-1/2 place-items-center rounded-full border p-0.5 text-sm opacity-0 transition group-hover:opacity-100 hover:scale-110 focus-visible:opacity-100";
</script>

<template>
    <div
        :class="[
            'relative text-xl transition-all',
            { 'ring-4 ring-red-200 select-none ring-inset': deleteMode },
        ]"
        v-bind="containerProps"
        @pointerdown="handlePointerDown"
        @pointermove="handlePointerMove"
        @pointerleave="hovered = null"
        @scroll="handleScroll"
    >
        <div
            class="divide-border-1 flex flex-col divide-y"
            v-bind="wrapperProps"
        >
            <div
                v-for="s in list"
                :key="s.data.uid"
                :data-segment-uid="s.data.uid"
                class="group relative"
                style="min-height: 80px"
            >
                <TranscriptionSegmentEditor
                    :ref="
                        (el) => {
                            if (el && segmentelements) {
                                segmentelements[s.index] =
                                    el as ComponentPublicInstance;
                            }
                        }
                    "
                    :segment="s.data"
                    :focused="focusedSegment?.uid === s.data.uid"
                    :in-range="preview?.uids.has(s.data.uid) ?? false"
                    :range-marks="preview?.deleted ?? false"
                    @word-focus="(w, seg) => $emit('wordFocus', w, seg)"
                    @update="handleSegmentUpdate(s.data.uid, $event)"
                    @toggle-delete="handleSegmentToggleDelete(s.data.uid)"
                    @focus-previous="focusSegment(s.index, -1)"
                    @focus-next="focusSegment(s.index, 1)"
                />
                <button
                    v-if="!deleteMode && s.index === 0"
                    :class="[addButtonClass, 'top-0 -translate-y-1/2']"
                    :title="$t('transcription.addSegmentBefore')"
                    @click="handleAddSegment(-1)"
                >
                    <Icon name="tabler:plus" />
                </button>
                <button
                    v-if="!deleteMode"
                    :class="[addButtonClass, 'bottom-0 translate-y-1/2']"
                    :title="$t('transcription.addSegment')"
                    @click="handleAddSegment(s.index)"
                >
                    <Icon name="tabler:plus" />
                </button>
            </div>
        </div>
    </div>
</template>
