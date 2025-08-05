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
        class="flex divide-x divide-neutral-300 rounded-lg border border-neutral-300 shadow-sm dark:divide-neutral-700 dark:border-neutral-700"
    >
        <button
            class="flex-1 rounded-l-lg bg-white px-3 py-1.5 text-sm dark:bg-neutral-800"
            @click="download"
        >
            {{ $t("transcription.download") }}
        </button>
        <select
            v-model="format"
            class="flex-1 rounded-r-lg bg-white px-3 py-1.5 text-sm text-neutral-400 transition-all duration-200 ease-out dark:bg-neutral-800"
            :style="{ maxWidth: widths[format] }"
        >
            <option value="json">JSON</option>
            <option value="srt">SRT</option>
            <option value="srt-words">SRT (words)</option>
        </select>
    </div>
</template>
