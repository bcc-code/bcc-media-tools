<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";

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

const deleteMode = ref(false);
</script>

<template>
    <div class="flex flex-col overflow-auto bg-black text-xl">
        <div class="flex gap-4 bg-slate-800 p-4">
            <slot name="actions"></slot>
            <BccButton
                class="ml-auto"
                @click="deleteMode = !deleteMode"
                :variant="!deleteMode ? 'primary' : 'secondary'"
            >
                Delete mode
            </BccButton>
        </div>
        <div class="flex flex-col overflow-auto">
            <template v-for="(s, index) in transcription.segments" :key="s.id">
                <SegmentEditor
                    v-if="!deleteMode"
                    class="py-2"
                    :segment="s"
                    :deleted="deletedIndexes.includes(index.toString())"
                    @word-focus="$emit('wordFocus', $event)"
                    @update="handleSegmentUpdate(index, $event)"
                    @toggle-delete="handleSegmentToggleDelete(index)"
                />
                <SegmentEditor
                    v-else
                    :segment="s"
                    :deleted="deletedIndexes.includes(index.toString())"
                    @click.stop="handleSegmentToggleDelete(index)"
                    class="cursor-pointer py-2 transition hover:bg-slate-900"
                />
            </template>
        </div>
    </div>
</template>
