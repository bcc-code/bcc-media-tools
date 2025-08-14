<script lang="ts" setup>
const props = defineProps<{
    segments: Segment[];
    filename: string;
}>();

const format = ref<"json" | "srt" | "srt-words">("json");

const toast = useToast();
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

    toast.add({
        icon: "heroicons:check",
        title: "Transcript downloaded successfully",
        color: "success",
    });
};

const widths = {
    json: "5.5rem",
    "srt-words": "20rem",
    srt: "5rem",
};
</script>

<template>
    <div
        class="divide-accented border-accented flex divide-x rounded-lg border shadow-sm"
    >
        <button
            class="bg-default flex-1 rounded-l-lg px-3 py-1.5 text-sm"
            @click="download"
        >
            {{ $t("transcription.download") }}
        </button>
        <select
            v-model="format"
            class="bg-default text-muted flex-1 rounded-r-lg px-3 py-1.5 text-sm transition-all duration-200 ease-out"
            :style="{ maxWidth: widths[format] }"
        >
            <option value="json">JSON</option>
            <option value="srt">SRT</option>
            <option value="srt-words">SRT (words)</option>
        </select>
    </div>
</template>
