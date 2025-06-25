<script lang="ts" setup>
import { BccButton, BccSelect } from "@bcc-code/design-library-vue";

const props = defineProps<{
    segments: Segment[];
    filename: string;
}>();

const format = ref<"json" | "srt" | "srt-words">("json");

const download = () => {
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

const widths = {
    json: "5.5rem",
    "srt-words": "20rem",
    srt: "5rem",
};
</script>

<template>
    <div
        class="flex divide-x divide-gray-300 rounded-lg border border-gray-300 shadow-sm"
    >
        <button
            class="flex-1 rounded-l-lg bg-white px-4 py-2"
            @click="download"
        >
            Download
        </button>
        <select
            v-model="format"
            class="flex-1 rounded-r-lg bg-white px-4 py-2 text-gray-400 transition-all duration-200 ease-out"
            :style="{ maxWidth: widths[format] }"
        >
            <option value="json">JSON</option>
            <option value="srt">SRT</option>
            <option value="srt-words">SRT (words)</option>
        </select>
    </div>
</template>
