<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";

const route = useRoute("transcription-id");

const loading = ref(true);

const reload = async () => {
    loading.value = true;
    const result = (await $fetch(`/api/vx/${route.params.id}/preview`)) as any;

    setTranscription(result);

    localStorage["ts-" + route.params.id] = JSON.stringify(result);

    return result;
};

const setTranscription = (result: any) => {
    transcription.value = result.transcription;
    segments.value = transcription.value?.segments!;
    video.value = result.video;
    fileName.value = result.filename;
    loading.value = false;
};

onMounted(async () => {
    const key = "ts-" + route.params.id;
    const saved = localStorage[key];

    if (saved) {
        setTranscription(JSON.parse(saved));
    }
});

const transcription = ref<TranscriptionResult>();

const fileName = ref<string>("untitled");

const segments = ref<Segment[]>([]);

const video = ref<string>();

const videoelement = ref<HTMLVideoElement>();

const handleWordFocus = (word: Word) => {
    const el = videoelement.value as HTMLVideoElement;
    if (!el) {
        return;
    }
    el.fastSeek(word.start);
};

watch(segments, () => {
    localStorage["ts-" + route.params.id] = JSON.stringify({
        transcription: {
            text: segments.value.map((s) => s.text).join(" "),
            segments: segments.value,
        },
        video: video.value,
        filename: fileName.value,
    });
});
</script>

<template>
    <div class="flex h-screen divide-x-2 divide-slate-500">
        <div class="flex w-1/2 flex-col">
            <div class="flex gap-4 bg-slate-900 p-4">
                <BccButton
                    @click="() => downloadTranscription(segments, fileName)"
                    >Download</BccButton
                >
                <BccButton @click="reload">Reload</BccButton>
                <p class="my-auto">Edits are saved locally</p>
            </div>
            <div v-if="loading" class="mx-auto animate-ping">Loading...</div>
            <TranscriptionEditor
                class="overflow-auto"
                v-if="transcription && !loading"
                :transcription="transcription"
                :file-name="fileName!"
                v-model="segments"
                @word-focus="handleWordFocus"
            />
        </div>
        <div class="flex w-1/2">
            <div class="m-auto">
                <video ref="videoelement" v-if="video" :src="video" controls />
            </div>
        </div>
    </div>
</template>
