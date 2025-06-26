<script lang="ts" setup>
import type { ComponentPublicInstance } from "vue";

const props = defineProps<{
    fileName?: string;
    transcription?: TranscriptionResult;
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
    return next?.start >= curr?.end + 1;
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
</script>

<template>
    <div
        :class="[
            'relative flex flex-col overflow-auto text-xl transition-all',
            { 'ring-4 ring-inset ring-red-200': deleteMode },
        ]"
    >
        <TransitionGroup
            v-if="transcription"
            tag="div"
            class="flex flex-col divide-y overflow-auto"
            enter-active-class="transition duration-300 ease-out"
            enter-from-class="opacity-0 scale-95"
            enter-to-class="opacity-100 scale-100"
            leave-active-class="transition duration-300 ease-out absolute"
            leave-from-class="opacity-100 scale-100"
            leave-to-class="opacity-0 scale-95"
            move-class="transition duration-300 ease-out"
        >
            <template
                v-for="(s, index) in transcription.segments"
                :key="`segment:${s.id}:${s.start}:${s.end}`"
            >
                <TranscriptionSegmentEditor
                    :ref="
                        (el) => {
                            if (el && segmentelements) {
                                segmentelements[index] =
                                    el as ComponentPublicInstance;
                            }
                        }
                    "
                    :segment="s"
                    :deleted="deletedIndexes.includes(index.toString())"
                    @word-focus="(w, s) => $emit('wordFocus', w, s)"
                    @update="handleSegmentUpdate(index, $event)"
                    @toggle-delete="handleSegmentToggleDelete(index)"
                    @focus-previous="focusSegment(index, -1)"
                    @focus-next="focusSegment(index, 1)"
                />
                <div
                    v-if="canAddSegment(index)"
                    :key="`segment:${s.id}:add`"
                    class="relative w-full"
                >
                    <button
                        class="absolute right-1/2 z-10 grid aspect-square size-6 -translate-y-1/2 place-items-center rounded-full bg-gray-200 p-0.5 text-sm hover:bg-gray-300"
                        :title="$t('transcription.addSegment')"
                        @click="addNewSegmentAt(index + 1)"
                    >
                        <Icon name="heroicons:plus-16-solid" />
                    </button>
                </div>
            </template>
        </TransitionGroup>
    </div>
</template>
