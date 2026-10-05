<script setup lang="ts">
import type { FlatSegment, FlatWord } from "~/utils/shortsTranscript";

const props = defineProps<{
    segments: FlatSegment[];
    currentTime: number;
    duration: number;
    start: number;
    end: number;
}>();

const emit = defineEmits<{
    seek: [time: number];
    setRange: [start: number, end: number];
}>();

const { t } = useI18n();

const flattened = computed(() => props.segments);
const wordsByIndex = computed(() => wordIndex(flattened.value));

// --- Clip highlight --------------------------------------------------------

/*
 * The highlight is derived from the clip range, not from the drag that may have
 * produced it, so trimming on the timeline or pressing I/O lights up the same
 * words. A drag writes the range as it goes and reads its own highlight back
 * out, which keeps one source of truth for what the clip is.
 */
const clipRange = computed(() =>
    clipHighlightRange(props.start, props.end, props.duration),
);

const isSelected = (word: FlatWord) =>
    clipRange.value != undefined && overlapsRange(word, clipRange.value);

// --- Selecting by dragging over the text -----------------------------------

const anchor = ref<number>();
const head = ref<number>();
const selecting = ref(false);

function emitRange() {
    if (anchor.value == undefined || head.value == undefined) return;
    const range = rangeFromSelection(
        wordsByIndex.value,
        anchor.value,
        head.value,
        props.duration,
    );
    if (range) emit("setRange", range.start, range.end);
}

function onWordDown(word: FlatWord) {
    anchor.value = word.index;
    head.value = word.index;
    selecting.value = true;
}

// Nothing is written until the drag reaches a second word, so a plain click
// stays a seek and leaves the clip the user already had alone.
function onWordEnter(word: FlatWord) {
    if (!selecting.value || anchor.value == undefined) return;
    head.value = word.index;
    if (head.value !== anchor.value) emitRange();
}

function onPointerUp() {
    if (!selecting.value) return;
    selecting.value = false;

    const clicked =
        anchor.value != undefined && anchor.value === head.value
            ? wordsByIndex.value.get(anchor.value)
            : undefined;

    // A click seeks where it landed; a drag seeks to the top of the new clip.
    if (clicked) emit("seek", clicked.start);
    else emit("seek", props.start);
}

useEventListener("pointerup", onPointerUp);
useEventListener("pointercancel", () => {
    selecting.value = false;
});

// --- Search ----------------------------------------------------------------

const query = ref("");
const matches = computed(() => matchingSegments(flattened.value, query.value));
const matchCursor = ref(0);
watch(matches, () => {
    matchCursor.value = 0;
});

const isMatch = (index: number) => matches.value.includes(index);

function gotoMatch(step: number) {
    if (matches.value.length === 0) return;
    matchCursor.value =
        (matchCursor.value + step + matches.value.length) %
        matches.value.length;
    const index = matches.value[matchCursor.value];
    if (index == undefined) return;
    scrollSegmentIntoView(index);
    const segment = flattened.value[index];
    if (segment) emit("seek", segment.start);
}

// --- Follow-along ----------------------------------------------------------

const rows = useTemplateRef<HTMLElement[]>("rows");

function scrollSegmentIntoView(index: number) {
    rows.value
        ?.find((el) => Number(el.dataset.index) === index)
        ?.scrollIntoView({ block: "center", behavior: "smooth" });
}

const activeSegment = computed(() =>
    flattened.value.find(
        (s) => props.currentTime >= s.start && props.currentTime < s.end,
    ),
);

/*
 * Scrolling with the playhead is only welcome while the reader is not steering.
 * Rather than trying to tell a programmatic scroll from a real one, note the
 * gestures that can only come from the user and leave them alone for a few
 * seconds afterwards.
 */
const FOLLOW_PAUSE_MS = 3000;
const lastInteraction = ref(0);
const noteInteraction = () => {
    lastInteraction.value = Date.now();
};

watch(
    () => activeSegment.value?.index,
    (index) => {
        if (index == undefined || selecting.value) return;
        if (Date.now() - lastInteraction.value < FOLLOW_PAUSE_MS) return;
        scrollSegmentIntoView(index);
    },
);

/*
 * Bring the clip into view when the range is set from outside the panel —
 * trimming on the timeline, or pressing I/O — because the words that just lit
 * up are easy to miss off-screen. A drag inside the panel is already looking at
 * the right place, so it is left alone, and unlike follow-along this answers a
 * deliberate action rather than playback, so it ignores the interaction pause.
 */
watch(
    () => [props.start, props.end],
    ([start, end], [prevStart, prevEnd]) => {
        if (selecting.value) return;
        if (start === prevStart && end === prevEnd) return;

        const range = clipRange.value;
        if (!range) return;

        // Follow whichever edge moved, so setting the out point shows the end of
        // the clip instead of jumping back to its start.
        const movedEndOnly = end !== prevEnd && start === prevStart;
        const inRange = flattened.value.filter((segment) =>
            overlapsRange(segment, range),
        );
        const target = movedEndOnly ? inRange.at(-1) : inRange.at(0);
        if (target) scrollSegmentIntoView(target.index);
    },
);

const formatStamp = (seconds: number) => {
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${String(s).padStart(2, "0")}`;
};
</script>

<template>
    <div class="flex min-h-0 flex-col gap-2">
        <div class="flex items-center gap-2">
            <DesignInput
                v-model="query"
                class="flex-1"
                :placeholder="t('shorts.generation.transcriptSearch')"
                @keydown.enter.prevent="gotoMatch(1)"
                @keydown.shift.enter.prevent="gotoMatch(-1)"
            />
            <span
                v-if="query.trim()"
                class="text-text-hint shrink-0 text-xs tabular-nums"
            >
                {{ matches.length ? matchCursor + 1 : 0 }}/{{ matches.length }}
            </span>
        </div>

        <p class="text-text-hint text-xs">
            {{ t("shorts.generation.transcriptHint") }}
        </p>

        <div
            class="border-border-1 min-h-0 flex-1 overflow-x-hidden overflow-y-auto rounded-xl border p-2"
            @wheel="noteInteraction"
            @pointerdown="noteInteraction"
        >
            <div
                v-for="segment in flattened"
                :key="segment.index"
                ref="rows"
                :data-index="segment.index"
                :class="[
                    'flex items-baseline gap-2 rounded-lg px-2 py-1.5',
                    {
                        'bg-surface-indent':
                            activeSegment?.index === segment.index,
                        'ring-semantic-info/40 ring-1': isMatch(segment.index),
                    },
                ]"
            >
                <button
                    type="button"
                    class="text-text-hint hover:text-text-default shrink-0 text-xs tabular-nums"
                    @click="emit('seek', segment.start)"
                >
                    {{ formatStamp(segment.start) }}
                </button>
                <!--
                    The trailing space is written as an expression, and lives
                    inside the span rather than between spans, for two reasons:
                    Vue's `condense` whitespace handling strips whitespace-only
                    text nodes between elements, so without it the words render
                    glued together and the line has nowhere to break; and
                    keeping it inside the highlight is what makes a selected
                    passage read as one band instead of striped words.
                -->
                <p class="text-body-3 min-w-0 flex-1 select-none">
                    <span
                        v-for="word in segment.words"
                        :key="word.index"
                        :class="[
                            'cursor-text',
                            isSelected(word)
                                ? 'bg-primary-default/30 dark:bg-primary-default/50'
                                : 'hover:bg-surface-indent',
                        ]"
                        @pointerdown.prevent="onWordDown(word)"
                        @pointerenter="onWordEnter(word)"
                        >{{ word.text }}{{ " " }}</span
                    >
                </p>
            </div>
        </div>
    </div>
</template>
