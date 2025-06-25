<script lang="ts" setup>
import { BccButton, BccToggle } from "@bcc-code/design-library-vue";
import type { ComponentPublicInstance } from "vue";

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({
        id: "transcription_index",
        title: "transcription",
    });
});

useHead({
    title: "Transcription",
});

const api = useAPI();

const route = useRoute("transcription-id");
const key = "ts-" + route.params.id;

const loading = ref(true);
const error = ref<string | null>(null);

const transcription = ref<TranscriptionResult>();

const fileName = ref<string>("untitled");

const segments = ref<Segment[]>([]);

const video = ref<string>();

const videoelement = ref<HTMLVideoElement>();

const segmentelements = ref<{
    [key: number]: Element | ComponentPublicInstance;
}>({});

function formatErrorMessage(msg: string | null): string | null {
    if (!msg) return null;
    // Remove [unknown] or similar prefix
    msg = msg.replace(/^\[.*?\]\s*/, "");
    // Capitalize first letter
    msg = msg.charAt(0).toUpperCase() + msg.slice(1);
    return msg;
}

const reload = async () => {
    loading.value = true;
    error.value = null;
    try {
        let result = await api.getTranscription({ VXID: route.params.id });
        setTranscription(result);
        localStorage[key] = JSON.stringify(result);
        return result;
    } catch (e: any) {
        error.value = e?.message || e?.toString() || "Unknown error";
        loading.value = false;
        transcription.value = undefined;
        segments.value = [];
        return null;
    }
};

const setTranscription = (result: any) => {
    transcription.value = result;
    console.log(transcription.value);
    console.log(segments.value);
    segments.value = transcription.value?.segments!;
    loading.value = false;
};

onMounted(async () => {
    const saved = localStorage[key];
    error.value = null;
    try {
        video.value = (await api.getPreview({ VXID: route.params.id })).url;
    } catch (e: any) {
        error.value = e?.message || e?.toString() || "Unknown error";
        loading.value = false;
        return;
    }
    fileName.value = key;

    if (saved) {
        setTranscription(JSON.parse(saved));
    } else {
        await reload();
    }
});

watch(videoelement, (el) => {
    if (el) {
        el.onseeked = () => {
            const current = el.currentTime;
            let index: number | null = null;

            let prev = 0;
            for (let i = 0; i < segments.value.length; i++) {
                const s = segments.value[i];
                if ((s.start < current || prev < current) && s.end > current) {
                    index = i;
                    break;
                }
                prev = s.end;
            }

            if (!index) return;

            const segmentElement = (
                segmentelements.value[index] as ComponentPublicInstance
            ).$el as HTMLDivElement;

            segmentElement.scrollIntoView({
                behavior: "smooth",
                block: "nearest",
                inline: "nearest",
            });
        };
    }
});

const handleWordFocus = (word: Word) => {
    const el = videoelement.value as HTMLVideoElement;
    if (!el) {
        return;
    }
    const seek = (localStorage.seekOnFocus ?? "true") === "true";
    if (seek) {
        if (el.fastSeek) {
            el.fastSeek(word.start);
        } else {
            el.currentTime = word.start;
        }
    }
};

watch(segments, () => {
    localStorage[key] = JSON.stringify({
        text: segments.value.map((s) => s.text).join(" "),
        segments: segments.value,
        video: video.value,
        filename: fileName.value,
    });
});

const seekOnFocus = computed({
    get() {
        return (localStorage.seekOnFocus ?? "true") === "true";
    },
    set(v) {
        localStorage.seekOnFocus = v ? "true" : "false";
    },
});
</script>

<template>
    <div class="flex h-screen divide-x-2 divide-neutral-500">
        <div class="flex w-1/2 flex-col">
            <div v-if="loading" class="mx-auto animate-ping">Loading...</div>
            <div v-if="error && !loading" class="mx-auto text-red-600 text-lg">{{ formatErrorMessage(error) }}</div>
            <TranscriptionEditor
                class="overflow-auto"
                v-if="transcription && !loading"
                :transcription="transcription"
                :file-name="fileName!"
                v-model="segments"
                v-model:segmentelements="segmentelements"
                @word-focus="handleWordFocus"
            >
                <template #actions>
                    <div class="flex flex-grow gap-4">
                        <TranscriptionDownloader
                            :segments="segments"
                            :filename="fileName"
                        />
                        <BccButton @click="reload">Reload</BccButton>
                        <p class="my-auto">Edits are saved locally</p>
                        <div class="my-auto ml-auto">
                            <BccToggle
                                id="seekonfocus"
                                v-model="seekOnFocus"
                                was-toggled
                                label="Seek on focus"
                            />
                        </div>
                    </div>
                </template>
            </TranscriptionEditor>
        </div>
        <div class="flex w-1/2">
            <div class="m-auto">
                <video ref="videoelement" v-if="video" :src="video" controls />
            </div>
        </div>
    </div>
</template>
