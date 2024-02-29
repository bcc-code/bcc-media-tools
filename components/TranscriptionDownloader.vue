<script lang="ts" setup>
import { BccButton, BccSelect } from "@bcc-code/design-library-vue";

const props = defineProps<{
    segments: Segment[];
    filename: string;
}>();

const format = ref<"json" | "srt" | "srt-words">("json");

const download = () => {
    console.log(format.value);
    switch (format.value) {
        case "json":
            downloadTranscriptionJSON(props.segments, props.filename);
            break;
        case "srt-words":
            downloadTranscriptionSRT(props.segments, props.filename, true);
            break;
        case "srt":
            downloadTranscriptionSRT(props.segments, props.filename, false);
            break;
    }
};
</script>

<template>
    <div>
        <BccSelect v-model="format">
            <option value="json">JSON</option>
            <option value="srt">SRT</option>
            <option value="srt-words">SRT (words)</option>
        </BccSelect>
        <BccButton @click="download">Download</BccButton>
    </div>
</template>
