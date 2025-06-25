<script lang="ts" setup>
import { BccButton, BccToggle } from "@bcc-code/design-library-vue";
import { normalizeProps, useMachine } from "@zag-js/vue";
import * as splitter from "@zag-js/splitter";
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

const transcription = ref<TranscriptionResult>();

const fileName = ref<string>("untitled");

const segments = ref<Segment[]>([]);

const video = ref<string>();

const videoelement = ref<HTMLVideoElement>();

const segmentelements = ref<{
    [key: number]: ComponentPublicInstance;
}>({});

const reset = async () => {
    loading.value = true;
    let result = await api.getTranscription({ VXID: route.params.id });
    setTranscription(result);
    localStorage[key] = JSON.stringify(result);
    return result;
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

    video.value = (await api.getPreview({ VXID: route.params.id })).url;
    fileName.value = key;

    if (saved) {
        setTranscription(JSON.parse(saved));
    } else {
        await reset();
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

const seekOnFocus = useLocalStorage("seekOnFocus", true);

const { deleteMode } = useDeleteMode();

const showManual = ref(false);

// Splitter
const storedSplitterSize = useLocalStorage("splitterSize", [50, 50]);
const splitterService = useMachine(splitter.machine, {
    id: useId(),
    defaultSize: storedSplitterSize.value,
    panels: [
        { id: "left", minSize: 25 },
        { id: "right", minSize: 25 },
    ],
    onResizeEnd({ size }) {
        storedSplitterSize.value = size;
    },
});

const splitterApi = computed(() =>
    splitter.connect(splitterService, normalizeProps),
);
</script>

<template>
    <div class="flex h-screen flex-col">
        <div
            class="flex items-center justify-between gap-4 border-b border-gray-400 bg-primary px-6 py-3"
        >
            <div class="flex gap-3">
                <p>{{ $t("transcription.changesSavedLocally") }}</p>
                <button class="-m-3 p-3 text-gray-500 underline" @click="reset">
                    {{ $t("transcription.reset") }}
                </button>
            </div>
            <div class="flex items-center gap-4">
                <BccToggle
                    id="seekonfocus"
                    v-model="seekOnFocus"
                    was-toggled
                    label="Seek on focus"
                />
                <BccToggle v-model="deleteMode" label="Delete mode" />
                <TranscriptionDownloader
                    :segments="segments"
                    :filename="fileName"
                />
                <BccButton>{{ $t("transcription.sendToReview") }}</BccButton>
                <button
                    class="-mx-3 aspect-square p-3"
                    @click="() => (showManual = true)"
                >
                    <Icon
                        name="heroicons:question-mark-circle"
                        class="text-xl"
                    />
                </button>
            </div>
        </div>
        <div v-bind="splitterApi.getRootProps()" class="flex bg-white">
            <div
                v-bind="splitterApi.getPanelProps({ id: 'left' })"
                class="flex flex-col"
            >
                <Icon
                    v-if="loading"
                    name="svg-spinners:bars-rotate-fade"
                    class="m-auto text-2xl"
                />
                <TranscriptionEditor
                    class="overflow-auto"
                    v-if="transcription && !loading"
                    :transcription="transcription"
                    :file-name="fileName!"
                    v-model="segments"
                    v-model:segmentelements="segmentelements"
                    @word-focus="handleWordFocus"
                />
            </div>
            <div class="flex h-full items-center border-x px-1">
                <div
                    v-bind="
                        splitterApi.getResizeTriggerProps({ id: 'left:right' })
                    "
                />
            </div>
            <div
                v-bind="splitterApi.getPanelProps({ id: 'right' })"
                class="flex bg-gray-100"
            >
                <div class="m-auto">
                    <Icon
                        v-if="loading && !video"
                        name="svg-spinners:bars-rotate-fade"
                        class="text-2xl"
                    />
                    <video
                        v-if="video"
                        ref="videoelement"
                        :src="video"
                        controls
                    />
                </div>
            </div>
        </div>
        <TranscriptionManual v-model:open="showManual" />
    </div>
</template>

<style>
[data-scope="splitter"][data-part="resize-trigger"] {
    @apply h-16 w-2 rounded-full bg-gray-300;
}
</style>
