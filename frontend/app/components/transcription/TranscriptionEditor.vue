<script lang="ts" setup>
import type { ComponentPublicInstance } from "vue";

const props = defineProps<{
    focusedSegment?: Segment;
    mediaDuration?: number;
}>();

defineEmits<{
    wordFocus: [Word, Segment];
}>();

const segments = defineModel<Segment[]>({ required: true });

const handleSegmentUpdate = (uid: string, segment: Segment) => {
    segments.value = updateSegment(segments.value, uid, () => segment);
};

// Marks the row rather than removing it, so the list the user sees and the one
// we submit stay the same shape. `toTranscription()` drops marked rows.
const handleSegmentToggleDelete = (uid: string) => {
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
            { 'ring-4 ring-red-200 ring-inset': deleteMode },
        ]"
        v-bind="containerProps"
    >
        <div
            class="divide-border-1 flex flex-col divide-y"
            v-bind="wrapperProps"
        >
            <div
                v-for="s in list"
                :key="s.data.uid"
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
