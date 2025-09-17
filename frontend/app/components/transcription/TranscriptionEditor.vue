<script lang="ts" setup>
import type { ComponentPublicInstance } from "vue";

const props = defineProps<{
    fileName?: string;
    transcription?: TranscriptionResult;
    focusedSegment?: Segment;
}>();

const emit = defineEmits<{
    wordFocus: [Word, Segment];
    updateSegments: [Segment[]];
}>();

const segments = defineModel<Segment[]>({ required: true });

const deletedIndexes = ref<string[]>([]);

const handleSegmentToggleDelete = (index: number) => {
    if (deletedIndexes.value.includes(index.toString())) {
        deletedIndexes.value = deletedIndexes.value.filter(
            (i) => i !== index.toString(),
        );
    } else {
        deletedIndexes.value = [...deletedIndexes.value, index.toString()];
    }
    segments.value =
        props.transcription?.segments.filter(
            (_, i) => !deletedIndexes.value.includes(i.toString()),
        ) ?? [];
};

const handleSegmentUpdate = (index: number, segment: Segment) => {
    const arr = [...(segments.value.map((s) => ({ ...s })) || [])];
    arr[index] = segment;
    segments.value = arr;
};

const { deleteMode } = useDeleteMode();

const segmentelements = defineModel<{
    [key: number]: ComponentPublicInstance;
}>("segmentelements", { default: {} });

function focusSegment(index: number, direction: number) {
    const next = segmentelements.value?.[index + direction];
    if (!next) return;
    const child = next.$el.querySelector(
        "[contenteditable]",
    ) as HTMLSpanElement;
    child.focus();
}

function canAddSegment(index: number) {
    if (deleteMode.value) return false;
    const curr = segments.value?.[index];
    const next = segments.value?.[index + 1];
    if (!curr || !next) return false;
    return next.start >= curr.end + 1;
}

function addNewSegmentAt(index: number) {
    const arr = [...(segments.value.map((s) => ({ ...s })) || [])];
    const prev = arr.at(index - 1);
    const next = arr.at(index);
    if (!prev || !next) return;

    arr.splice(index, 0, {
        id: (prev.id + next.id) / 2,
        seek: 0,
        start: prev.end,
        end: next.start,
        text: "...",
        tokens: [],
        temperature: 0,
        avg_logprob: 0,
        compression_ration: 0,
        no_speech_prob: 0,
        confidence: 0,
        words: [
            {
                text: "...",
                start: prev.end,
                end: next.start,
                confidence: 0,
            },
        ],
    });

    emit("updateSegments", arr);
}

const transcriptionSegments = computed(
    () => props.transcription?.segments ?? [],
);
const { list, wrapperProps, containerProps } = useVirtualList(
    transcriptionSegments,
    {
        itemHeight: 80,
        overscan: 10,
    },
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
        <TransitionGroup
            v-if="list"
            tag="div"
            class="divide-default flex flex-col divide-y"
            enter-active-class="transition duration-300 ease-out"
            enter-from-class="opacity-0 scale-95"
            enter-to-class="opacity-100 scale-100"
            leave-active-class="transition duration-300 ease-out absolute"
            leave-from-class="opacity-100 scale-100"
            leave-to-class="opacity-0 scale-95"
            move-class="transition duration-300 ease-out"
            v-bind="wrapperProps"
        >
            <template
                v-for="s in list"
                :key="`segment:${s.index}:${s.data.id}:${s.data.start}:${s.data.end}`"
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
                    :focused="focusedSegment == s.data"
                    :deleted="deletedIndexes.includes(s.index.toString())"
                    style="min-height: 80px"
                    @word-focus="(w, s) => $emit('wordFocus', w, s)"
                    @update="handleSegmentUpdate(s.index, $event)"
                    @toggle-delete="handleSegmentToggleDelete(s.index)"
                    @focus-previous="focusSegment(s.index, -1)"
                    @focus-next="focusSegment(s.index, 1)"
                />
                <div
                    v-if="canAddSegment(s.index)"
                    :key="`segment:${s.index}:${s.data.id}:add`"
                    class="relative w-full"
                >
                    <button
                        class="bg-accented absolute right-1/2 z-10 grid aspect-square size-6 -translate-y-1/2 place-items-center rounded-full p-0.5 text-sm hover:scale-110"
                        :title="$t('transcription.addSegment')"
                        @click="addNewSegmentAt(s.index + 1)"
                    >
                        <Icon name="heroicons:plus-16-solid" />
                    </button>
                </div>
            </template>
        </TransitionGroup>
    </div>
</template>
