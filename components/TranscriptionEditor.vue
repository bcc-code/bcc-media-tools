<script lang="ts" setup>
const props = defineProps<{
    fileName: string;
    transcription: TranscriptionResult;
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
    segments.value = props.transcription.segments.filter(
        (_, i) => !deletedIndexes.value.includes(i.toString()),
    );
};

const handleSegmentUpdate = (index: number, segment: Segment) => {
    const arr = [...(segments.value.map((s) => ({ ...s })) || [])];
    arr[index] = segment;
    segments.value = arr;
};
</script>

<template>
    <div class="flex flex-col gap-4 overflow-auto bg-black p-4 text-xl">
        <SegmentEditor
            v-for="(s, index) in transcription.segments"
            :key="s.id"
            :segment="s"
            :deleted="deletedIndexes.includes(index.toString())"
            @word-focus="$emit('wordFocus', $event)"
            @update="handleSegmentUpdate(index, $event)"
            @toggle-delete="handleSegmentToggleDelete(index)"
        />
    </div>
</template>
