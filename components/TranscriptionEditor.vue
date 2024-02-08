<script lang="ts" setup>
defineProps<{
    fileName: string;
    transcription: TranscriptionResult;
}>();

defineEmits<{
    wordFocus: [Word];
}>();

const segments = defineModel<Segment[]>({ required: true });

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
            :key="index"
            :segment="s"
            @word-focus="$emit('wordFocus', $event)"
            @update="handleSegmentUpdate(index, $event)"
        />
    </div>
</template>
