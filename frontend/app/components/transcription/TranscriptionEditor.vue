<script lang="ts" setup>
import type { ComponentPublicInstance } from "vue";

defineProps<{
    focusedSegment?: Segment;
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
    segments.value = insertSegmentAfter(segments.value, index);
};

const { deleteMode } = useDeleteMode();

const canAdd = (index: number) =>
    !deleteMode.value && canInsertAfter(segments.value, index);

const segmentelements = defineModel<Record<number, ComponentPublicInstance>>(
    "segmentelements",
    { default: () => ({}) },
);

function focusSegment(index: number, direction: number) {
    const next = segmentelements.value?.[index + direction];
    if (!next) return;
    const child = next.$el.querySelector(
        "[contenteditable]",
    ) as HTMLSpanElement | null;
    child?.focus();
}

const { list, wrapperProps, containerProps } = useVirtualList<Segment>(
    computed(() => segments.value),
    { itemHeight: 80, overscan: 10 },
);
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
            <template v-for="s in list" :key="s.data.uid">
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
                    style="min-height: 80px"
                    @word-focus="(w, seg) => $emit('wordFocus', w, seg)"
                    @update="handleSegmentUpdate(s.data.uid, $event)"
                    @toggle-delete="handleSegmentToggleDelete(s.data.uid)"
                    @focus-previous="focusSegment(s.index, -1)"
                    @focus-next="focusSegment(s.index, 1)"
                />
                <div
                    v-if="canAdd(s.index)"
                    :key="`${s.data.uid}:add`"
                    class="relative w-full"
                >
                    <button
                        class="bg-surface-raise border-border-1 absolute right-1/2 z-10 grid aspect-square size-6 -translate-y-1/2 place-items-center rounded-full border p-0.5 text-sm hover:scale-110"
                        :title="$t('transcription.addSegment')"
                        @click="handleAddSegment(s.index)"
                    >
                        <Icon name="tabler:plus" />
                    </button>
                </div>
            </template>
        </div>
    </div>
</template>
