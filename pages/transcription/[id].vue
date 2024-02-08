<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";

const route = useRoute("transcription-id");

const loading = ref(true);

onMounted(async () => {
    const result = (await $fetch(`/api/vx/${route.params.id}/preview`)) as any;

    transcription.value = result.transcription;
    segments.value = transcription.value?.segments!;
    video.value = result.video;
    loading.value = false;
});

const transcription = ref<TranscriptionResult>();

const fileName = ref<string>();

const segments = ref<Segment[]>([]);

const video = ref<string>();

const videoelement = ref<HTMLVideoElement>();

const handleWordFocus = (word: Word) => {
    const el = videoelement.value as HTMLVideoElement;
    console.log("AHWAIUWH");
    if (!el) {
        return;
    }
    el.fastSeek(word.start);
};
</script>

<template>
    <div class="flex h-screen">
        <div class="flex flex-col">
            <div class="flex gap-4">
                <BccButton
                    @click="() => downloadTranscription(segments, 'eeee')"
                    >Download</BccButton
                >
            </div>
            <div v-if="loading" class="mx-auto animate-ping">Loading...</div>
            <TranscriptionEditor
                class="overflow-auto"
                v-if="transcription"
                :transcription="transcription"
                :file-name="fileName!"
                v-model="segments"
                @word-focus="handleWordFocus"
            />
        </div>
        <div>
            <video ref="videoelement" v-if="video" :src="video" controls />
        </div>
    </div>
</template>
