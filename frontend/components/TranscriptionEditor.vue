<script lang="ts" setup>
import type { ComponentPublicInstance } from "vue";

const props = defineProps<{
    fileName?: string;
    transcription?: TranscriptionResult;
}>();

defineEmits<{
    wordFocus: [Word];
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
    [key: number]: Element | ComponentPublicInstance;
}>("segmentelements");
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
            <SegmentEditor
                v-for="(s, index) in transcription.segments"
                :key="s.id"
                :ref="
                    (el) => {
                        if (el && segmentelements) {
                            segmentelements[index] = el;
                        }
                    }
                "
                :segment="s"
                :deleted="deletedIndexes.includes(index.toString())"
                @word-focus="$emit('wordFocus', $event)"
                @update="handleSegmentUpdate(index, $event)"
                @toggle-delete="handleSegmentToggleDelete(index)"
            />
        </TransitionGroup>
    </div>
</template>
