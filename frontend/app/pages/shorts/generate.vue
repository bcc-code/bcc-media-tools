<script setup lang="ts">
const route = useRoute("shorts-generate");
const vxId = computed(() => route.query.id?.toString());

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({
        id: "shorts_id",
        title: "shorts",
        meta: {
            id: vxId.value,
        },
    });
});

useHead({
    title: "Shorts generation",
});

const api = useAPI();
const base = useRuntimeConfig().public.grpcUrl;

const {
    data: videoUrl,
    status,
    error,
    refresh,
} = useAsyncData(
    () => `preview:${vxId.value}`,
    () => api.getShortsPreview({ VXID: vxId.value }),
    { transform: (data) => data.url },
);

// Assets without a preview shape are common enough that the error state offers
// the Cantemo action which creates one, for users whose permissions include it.
const { chips } = useCantemoActions(vxId);
const previewChip = computed(() => chips.value.find((c) => c.id === "preview"));

// Optional: the page is fully usable without it, so it loads on its own and
// never gates rendering.
const { data: transcriptSegments, status: transcriptStatus } = useAsyncData(
    () => `shorts-transcript:${vxId.value}`,
    () => api.getShortsTranscript({ VXID: vxId.value }),
    // Flattened here rather than in the panel because the page needs the same
    // result to decide whether there is anything worth showing.
    { transform: (data) => flattenSegments(data.segments ?? []) },
);

const videoElement = useTemplateRef("videoElement");

// The transcript sits in a grid cell next to the video. Grid rows size to their
// tallest child, so without an explicit bound the transcript sets the row height
// and grows the page instead of scrolling — pushing the timeline below the fold.
// Bounding it to the measured video height keeps both columns the same height.
const { width: videoBoxWidth, height: videoHeight } =
    useElementSize(videoElement);

// The 9:16 guide is derived from the source's real dimensions rather than an
// assumed 16:9, since the crop is defined in terms of the frame height.
const videoSize = ref({ width: 0, height: 0 });
const showCropGuide = useLocalStorage("shortsCropGuide", false);

const duration = ref<number | undefined>(0);
const startTime = ref<number | undefined>(0);
const endTime = ref<number | undefined>(0);

const shortDuration = computed(() => {
    if (startTime.value == undefined || endTime.value == undefined) return 0;
    return Math.ceil(endTime.value - startTime.value);
});

// The >60s rule is deliberately not part of this: it is editorial advice, not a
// constraint the pipeline enforces, so it stays a warning.
const isValidRange = computed(
    () =>
        startTime.value != undefined &&
        endTime.value != undefined &&
        endTime.value > startTime.value,
);

useEventListener(
    videoElement,
    "loadeddata",
    () => {
        const el = videoElement.value;
        duration.value = el?.duration;
        startTime.value = 0;
        endTime.value = duration.value;
        if (el?.videoWidth && el?.videoHeight) {
            videoSize.value = { width: el.videoWidth, height: el.videoHeight };
        }
    },
    { once: true },
);

const previewingShort = ref(false);
function previewShort() {
    if (
        !videoElement.value ||
        startTime.value == undefined ||
        endTime.value == undefined
    )
        return;

    videoElement.value.currentTime = startTime.value;
    videoElement.value.play();
    previewingShort.value = true;
}

const currentTime = ref(0);
function onSeek(time: number) {
    if (!videoElement.value) return;
    videoElement.value.currentTime = time;
    currentTime.value = time;
}
useEventListener(videoElement, "timeupdate", () => {
    if (!videoElement.value) return;
    currentTime.value = videoElement.value.currentTime;
    if (
        endTime.value != undefined &&
        currentTime.value >= endTime.value &&
        previewingShort.value
    ) {
        previewingShort.value = false;
        videoElement.value.pause();
    }
});

const colorMode = useColorMode();
const showManual = ref(false);
const hasUsedBefore = useLocalStorage("hasUsedShortsGeneration", false);
const manualGif = computed(() => {
    if (colorMode.value === "dark") {
        return "/images/gifs/shorts-generation-dark.gif";
    }
    return "/images/gifs/shorts-generation-light.gif";
});
onMounted(() => {
    if (!hasUsedBefore.value) {
        setTimeout(() => {
            showManual.value = true;
            hasUsedBefore.value = true;
        }, 1000);
    }
});

const zoom = ref(1);
const scrubber = useTemplateRef("scrubber");
const { width: scrubberWidth } = useElementSize(() => scrubber.value?.$el);

// Fit the whole video in the track until the user takes over the zoom —
// otherwise every resize refits and discards the level they set.
const zoomPinned = ref(false);
watch([duration, scrubberWidth], ([d, s]) => {
    if (zoomPinned.value || !d || !s) return;
    zoom.value = s / d;
});
function onZoomInput(value: number) {
    zoomPinned.value = true;
    zoom.value = value;
}

const toaster = useToast();
const confirmSubmit = ref(false);
const submitting = ref(false);
async function submit() {
    if (!isValidRange.value || submitting.value) return;
    submitting.value = true;
    try {
        await api.submitShort({
            VXID: vxId.value,
            InSeconds: startTime.value,
            OutSeconds: endTime.value,
        });
        toaster.create({
            title: "Short submitted successfully",
            type: "success",
        });
        confirmSubmit.value = false;
        navigateTo("/shorts");
    } catch (err) {
        toaster.create({
            title: "Failed to submit short",
            description: (err as Error)?.message,
            type: "error",
        });
    } finally {
        submitting.value = false;
    }
}

// Written continuously while a transcript drag is in progress, so it must not
// seek: the panel emits a single seek of its own once the drag ends.
function setRangeFromTranscript(from: number, to: number) {
    const max = duration.value ?? to;
    startTime.value = Math.max(0, Math.min(from, max));
    endTime.value = Math.max(startTime.value, Math.min(to, max));
}

function setStartPoint() {
    if (endTime.value == undefined) {
        startTime.value = currentTime.value;
        return;
    }
    startTime.value = Math.min(currentTime.value, endTime.value);
}

function setEndPoint() {
    if (startTime.value == undefined) {
        endTime.value = currentTime.value;
        return;
    }
    endTime.value = Math.max(currentTime.value, startTime.value);
}

const formattedDuration = (duration: number) => {
    const minutes = Math.floor(duration / 60);
    const seconds = duration % 60;
    return `${minutes}:${seconds < 10 ? "0" : ""}${seconds}`;
};

useVideoKeyboardControls({
    togglePlay: () => {
        if (videoElement.value) {
            videoElement.value.paused
                ? videoElement.value.play()
                : videoElement.value.pause();
        }
    },
    backward: () => {
        if (videoElement.value) {
            videoElement.value.currentTime -= 1;
        }
    },
    forward: () => {
        if (videoElement.value) {
            videoElement.value.currentTime += 1;
        }
    },
    setStartPoint,
    setEndPoint,
});
</script>

<template>
    <div class="mx-auto flex w-full max-w-7xl flex-col gap-4 p-8">
        <header class="mb-4 flex items-center justify-between">
            <div>
                <h1 class="text-heading-3 text-text-default">
                    {{ $t("shorts.generation.title") }}
                </h1>
                <p class="text-text-muted text-sm">
                    {{ $t("shorts.generation.description") }}
                </p>
            </div>
            <DesignButton
                icon="tabler:send"
                :disabled="!isValidRange || status !== 'success'"
                :loading="submitting"
                @click="confirmSubmit = true"
            >
                {{ $t("shorts.generation.submit") }}
            </DesignButton>
            <DesignDialog
                v-model:open="confirmSubmit"
                :title="$t('shorts.generation.submitConfirmationTitle')"
                :description="$t('shorts.generation.submitConfirmationMessage')"
            >
                <div class="flex w-full justify-end gap-2">
                    <DesignButton
                        variant="tertiary"
                        @click="confirmSubmit = false"
                    >
                        {{ $t("shorts.generation.submitConfirmationCancel") }}
                    </DesignButton>
                    <DesignButton
                        variant="primary"
                        :disabled="!isValidRange"
                        :loading="submitting"
                        @click="submit"
                    >
                        {{ $t("shorts.generation.submitConfirmationSubmit") }}
                    </DesignButton>
                </div>
            </DesignDialog>
        </header>
        <template v-if="status === 'success'">
            <div
                class="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]"
                :style="{
                    '--video-h': videoHeight ? `${videoHeight}px` : '60vh',
                }"
            >
                <div class="relative self-start">
                    <video
                        ref="videoElement"
                        :src="videoUrl"
                        controls
                        class="bg-surface-default aspect-video w-full shadow-xl"
                    />
                    <ShortsCropGuide
                        v-if="showCropGuide"
                        :element-width="videoBoxWidth"
                        :element-height="videoHeight"
                        :video-width="videoSize.width"
                        :video-height="videoSize.height"
                    />
                </div>

                <ShortsTranscriptPanel
                    v-if="transcriptSegments?.length"
                    :segments="transcriptSegments"
                    :current-time="currentTime"
                    :duration="duration ?? 0"
                    :start="startTime ?? 0"
                    :end="endTime ?? 0"
                    class="max-h-[60vh] lg:max-h-(--video-h)"
                    @seek="onSeek"
                    @set-range="setRangeFromTranscript"
                />
                <div
                    v-else-if="transcriptStatus === 'pending'"
                    class="max-h-[60vh] space-y-2 lg:max-h-(--video-h)"
                >
                    <DesignSkeleton class="h-9 w-full" />
                    <DesignSkeleton class="h-40 w-full" />
                </div>
                <DesignBanner
                    v-else-if="transcriptStatus === 'error'"
                    icon="tabler:alert-triangle"
                    variant="warning"
                    class="self-start"
                >
                    {{ $t("shorts.generation.transcriptFailed") }}
                </DesignBanner>
                <DesignBanner
                    v-else
                    icon="tabler:file-text-off"
                    variant="neutral"
                    class="self-start"
                >
                    {{ $t("shorts.generation.noTranscript") }}
                </DesignBanner>
            </div>
            <div class="flex items-center gap-2">
                <div class="tabular-nums">
                    <p
                        :class="[
                            'font-bold',
                            {
                                'text-red-600 dark:text-red-300':
                                    shortDuration > 60,
                            },
                        ]"
                    >
                        {{ formattedDuration(shortDuration) }}
                        <span
                            v-if="shortDuration > 60"
                            class="ml-1 inline-block origin-left font-normal opacity-50"
                        >
                            {{ $t("shorts.generation.durationWarning") }}
                        </span>
                    </p>
                    <p
                        v-if="startTime != undefined && endTime != undefined"
                        class="text-text-hint text-sm"
                    >
                        {{ formatTime(startTime) }} - {{ formatTime(endTime) }}
                    </p>
                </div>
                <DesignButton
                    class="border-border-1 ml-auto border"
                    variant="secondary"
                    @click="setStartPoint"
                >
                    {{ $t("shorts.generation.setStartPoint") }}
                    <span class="text-text-hint ml-1 text-xs">I</span>
                </DesignButton>
                <DesignButton
                    class="border-border-1 border"
                    variant="secondary"
                    @click="setEndPoint"
                >
                    {{ $t("shorts.generation.setEndPoint") }}
                    <span class="text-text-hint ml-1 text-xs">O</span>
                </DesignButton>
                <DesignButton
                    class="border-border-1 border"
                    variant="secondary"
                    @click="previewShort"
                >
                    {{ $t("shorts.generation.previewShort") }}
                </DesignButton>
                <DesignTooltip :content="$t('shorts.generation.cropGuideHint')">
                    <DesignButton
                        class="border-border-1 border"
                        :variant="showCropGuide ? 'primary' : 'secondary'"
                        @click="showCropGuide = !showCropGuide"
                    >
                        {{ $t("shorts.generation.cropGuide") }}
                    </DesignButton>
                </DesignTooltip>
            </div>
            <ShortsTimelineScrubber
                v-if="
                    duration != undefined &&
                    startTime != undefined &&
                    endTime != undefined
                "
                ref="scrubber"
                :min="0"
                :max="duration"
                :current="currentTime"
                :zoom="zoom"
                :vxid="vxId ?? ''"
                :base="base"
                v-model:start="startTime"
                v-model:end="endTime"
                @seek="onSeek"
            />
            <DesignSlider
                :model-value="zoom"
                :min="0.1"
                :max="10"
                :step="0.01"
                @update:model-value="onZoomInput"
            />
        </template>
        <div
            v-else-if="status === 'error'"
            class="flex flex-col items-center gap-4 py-16"
        >
            <Icon name="tabler:alert-triangle" class="text-text-hint size-10" />
            <p class="text-text-muted max-w-md text-center">
                {{ error?.message ?? $t("shorts.generation.previewFailed") }}
            </p>
            <p class="text-text-hint max-w-md text-center text-sm">
                {{ $t("shorts.generation.previewFailedHint") }}
            </p>
            <div class="flex gap-2">
                <DesignButton variant="secondary" @click="refresh()">
                    {{ $t("shorts.generation.retry") }}
                </DesignButton>
                <DesignButton
                    v-if="previewChip"
                    variant="secondary"
                    @click="previewChip.run()"
                >
                    {{ previewChip.label }}
                </DesignButton>
            </div>
        </div>

        <template v-else>
            <DesignSkeleton class="aspect-video w-full" />
            <div class="flex items-center gap-2">
                <div class="space-y-2">
                    <DesignSkeleton class="h-5 w-16" />
                    <DesignSkeleton class="h-4 w-48" />
                </div>
                <DesignSkeleton class="ml-auto h-8 w-28" />
                <DesignSkeleton class="h-8 w-28" />
                <DesignSkeleton class="h-8 w-28" />
            </div>
            <DesignSkeleton class="h-38 w-full" />
            <DesignSkeleton class="h-2 w-full" />
        </template>

        <DesignDialog v-model:open="showManual">
            <img :src="manualGif" class="w-full rounded-lg" />
        </DesignDialog>
    </div>
</template>
