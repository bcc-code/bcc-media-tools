<script lang="ts" setup>
const transcription = ref<TranscriptionResult>();

const fileName = ref<string>();

const handleFile = (event: Event) => {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (file) {
        fileName.value = file.name;
        const reader = new FileReader();
        reader.onload = (e) => {
            const result = e.target?.result;
            if (result) {
                transcription.value = JSON.parse(result.toString());
                transcription.value?.segments.forEach((s) => {
                    s.text = s.text.trim();
                    s.words = s.words.map((w) => ({
                        ...w,
                        text: w.text.trim(),
                    }));
                });

                segments.value = transcription.value?.segments;
            }
        };
        reader.readAsText(file);
    }
};

const segments = ref<Segment[]>();

const handleSegmentUpdate = (index: number, segment: Segment) => {
    const arr = [...(segments.value?.map((s) => ({ ...s })) || [])];
    arr[index] = segment;
    segments.value = arr;
};

const download = () => {
    const data = JSON.stringify({
        text: segments.value?.map((s) => s.text).join(" "),
        segments: segments.value,
    });
    const blob = new Blob([data], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = fileName.value + "-edited.json";
    a.click();
    URL.revokeObjectURL(url);
};
</script>

<template>
    <div>
        <div class="">
            <input
                type="file"
                placeholder="File here"
                accept="application/json"
                @input="handleFile"
            />
        </div>
        <button @click="download">Download</button>

        <div
            v-if="transcription"
            class="flex flex-col gap-4 bg-black p-4 text-xl"
        >
            <SegmentEditor
                v-for="(s, index) in transcription.segments"
                :key="index"
                :segment="s"
                @update="handleSegmentUpdate(index, $event)"
            />
        </div>
    </div>
</template>
