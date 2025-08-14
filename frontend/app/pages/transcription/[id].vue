<script lang="ts" setup>
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
const routeId = route.params.id;

const loading = ref(true);
const error = ref<string | null>(null);

const transcription = ref<TranscriptionResult>();

const fileName = ref<string>("untitled");

const segments = ref<Segment[]>([]);

const video = ref<string>();

const videoelement = ref<HTMLVideoElement>();

const segmentelements = ref<{
    [key: number]: ComponentPublicInstance;
}>({});

function formatErrorMessage(msg: string | null): string | null {
    if (!msg) return null;
    // Remove [unknown] or similar prefix
    msg = msg.replace(/^\[.*?\]\s*/, "");
    // Capitalize first letter
    msg = msg.charAt(0).toUpperCase() + msg.slice(1);
    return msg;
}

const { t } = useI18n();
const toast = useToast();
const reset = async (notify: boolean = true) => {
    loading.value = true;
    error.value = null;
    try {
        let result = await api.getTranscription({ VXID: routeId });
        setTranscription(result);
        localStorage[key] = JSON.stringify(result);
        if (notify) {
            toast.add({
                icon: "heroicons:check",
                title: t("transcription.resetSuccess"),
                color: "success",
            });
        }
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
    segments.value = transcription.value?.segments!;
    loading.value = false;
};

const loadingSubmit = ref(false);
const submitToMediabanken = async () => {
    loadingSubmit.value = true;
    try {
        await api.submitTranscription({
            VXID: routeId,
            transcription: transcription.value,
        });
        localStorage.removeItem(key);
        toast.add({
            icon: "heroicons:check",
            title: t("transcription.submitSuccess"),
            color: "success",
        });
        navigateTo("/transcription");
    } catch (err) {
        toast.add({
            icon: "heroicons:exclamation",
            title: t("transcription.submitError"),
            color: "error",
        });
        loadingSubmit.value = false;
    }
};

onMounted(async () => {
    const saved = localStorage[key];
    error.value = null;
    try {
        video.value = (await api.getPreview({ VXID: routeId })).url;
    } catch (e: any) {
        error.value = e?.message || e?.toString() || "Unknown error";
        loading.value = false;
        return;
    }
    fileName.value = key;

    if (saved) {
        setTranscription(JSON.parse(saved));
    } else {
        await reset(false);
    }
});

watch(videoelement, (el) => {
    if (el) {
        let prevIndex: number | null = null;
        el.ontimeupdate = () => {
            const current = el.currentTime;
            let index: number | null = null;

            let prev = 0;
            for (let i = 0; i < segments.value.length; i++) {
                const s = segments.value[i]!;
                if ((s.start < current || prev < current) && s.end > current) {
                    index = i;
                    break;
                }
                prev = s.end;
            }

            if (!index) return;
            if (index === prevIndex) return;

            focusedSegment.value = segments.value[index];

            const segmentElement = (
                segmentelements.value[index] as ComponentPublicInstance
            ).$el as HTMLDivElement;

            segmentElement.scrollIntoView({
                behavior: "smooth",
                block: "center",
            });

            prevIndex = index;
        };
    }
});

const focusedSegment = ref<Segment>();
const handleWordFocus = (word: Word, segment: Segment) => {
    focusedSegment.value = segment;

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
const previewSubtitles = useLocalStorage("previewSubtitles", true);
const { deleteMode } = useDeleteMode();

const showManual = ref(false);
// Show manual the first time the user opens the tool
const hasOpenedManual = useLocalStorage("hasOpenedManual", false);
onMounted(() => {
    if (!hasOpenedManual.value) {
        setTimeout(() => {
            showManual.value = true;
            hasOpenedManual.value = true;
        }, 1000);
    }
});

function setSegments(s: Segment[]) {
    segments.value = s;
    if (!transcription.value) return;
    transcription.value.segments = s;
}

const showSubmitConfirmationModal = ref(false);

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
    <div class="flex h-[calc(100dvh-var(--header-height))] flex-col">
        <div
            class="border-default bg-default flex items-center justify-between gap-4 border-b px-6 py-3"
        >
            <div class="flex gap-3">
                <p>{{ $t("transcription.changesSavedLocally") }}</p>
                <button
                    class="-m-3 p-3 text-neutral-500 underline"
                    @click="() => reset()"
                >
                    {{ $t("transcription.reset") }}
                </button>
            </div>
            <div class="flex items-center gap-4">
                <USwitch
                    v-model="previewSubtitles"
                    :label="$t('transcription.previewSubtitles')"
                />
                <USwitch
                    v-model="seekOnFocus"
                    was-toggled
                    :label="$t('transcription.seekOnFocus')"
                />
                <USwitch
                    v-model="deleteMode"
                    :label="$t('transcription.deleteMode')"
                />
                <LanguageSwitcher />
                <TranscriptionDownloader
                    :segments="segments"
                    :filename="fileName"
                />
                <UButton @click="showSubmitConfirmationModal = true">
                    {{ $t("transcription.save") }}
                </UButton>
                <button
                    class="-mx-3 aspect-square p-3"
                    @click="showManual = true"
                >
                    <Icon
                        name="heroicons:question-mark-circle"
                        class="text-xl"
                    />
                </button>
            </div>
        </div>
        <div v-bind="splitterApi.getRootProps()" class="bg-default flex">
            <div
                v-bind="splitterApi.getPanelProps({ id: 'left' })"
                class="flex flex-col"
            >
                <Icon
                    v-if="loading"
                    name="svg-spinners:bars-rotate-fade"
                    class="m-auto text-2xl"
                />
                <div
                    v-if="error && !loading"
                    class="mx-auto text-lg text-red-600"
                >
                    {{ formatErrorMessage(error) }}
                </div>
                <TranscriptionEditor
                    class="ml-auto w-full max-w-7xl overflow-auto"
                    v-if="transcription && !loading"
                    v-model="segments"
                    v-model:segmentelements="segmentelements"
                    :transcription="transcription"
                    :file-name="fileName!"
                    :focused-segment="focusedSegment"
                    @word-focus="handleWordFocus"
                    @update-segments="(s) => setSegments(s)"
                />
            </div>
            <div class="border-default flex h-full items-center border-x px-1">
                <div
                    v-bind="
                        splitterApi.getResizeTriggerProps({ id: 'left:right' })
                    "
                />
            </div>
            <div
                v-bind="splitterApi.getPanelProps({ id: 'right' })"
                class="flex flex-col bg-neutral-100 dark:bg-neutral-950"
            >
                <div class="relative m-auto p-4">
                    <Icon
                        v-if="loading && !video"
                        name="svg-spinners:bars-rotate-fade"
                        class="text-2xl"
                    />
                    <template v-if="video">
                        <video
                            ref="videoelement"
                            :src="video"
                            controls
                            class="bg-default shadow-xl"
                        />
                        <p
                            v-if="previewSubtitles && focusedSegment"
                            class="absolute bottom-16 left-1/2 w-max max-w-[75%] -translate-x-1/2 bg-black/50 p-2 text-center text-2xl text-white"
                        >
                            {{
                                focusedSegment.words
                                    .map((w) => w.text)
                                    .join(" ")
                            }}
                        </p>
                    </template>
                </div>
            </div>
        </div>
        <TranscriptionManual v-model:open="showManual" />
        <UModal
            v-model:open="showSubmitConfirmationModal"
            :close="false"
            :title="$t('transcription.submitConfirmationTitle')"
            :description="$t('transcription.submitConfirmationMessage')"
        >
            <template #footer>
                <UButton
                    variant="link"
                    autofocus
                    class="ml-auto"
                    @click="showSubmitConfirmationModal = false"
                >
                    {{ $t("transcription.submitConfirmationCancel") }}
                </UButton>
                <UButton :disabled="loadingSubmit" @click="submitToMediabanken">
                    <Icon
                        v-if="loadingSubmit"
                        name="svg-spinners:bars-rotate-fade"
                    />
                    {{ $t("transcription.submitConfirmationSubmit") }}
                </UButton>
            </template>
        </UModal>
    </div>
</template>

<style>
[data-scope="splitter"][data-part="resize-trigger"] {
    height: calc(var(--spacing) * 16);
    width: calc(var(--spacing) * 2);
    border-radius: calc(infinity * 1px);
    background-color: var(--ui-color-neutral-300);

    &:where(.dark, .dark *) {
        background-color: var(--ui-color-neutral-700);
    }
}
</style>
