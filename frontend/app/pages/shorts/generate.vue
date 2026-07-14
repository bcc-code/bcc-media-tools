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

const { data: videoUrl, status } = useAsyncData(
    () => `preview:${vxId.value}`,
    () => api.getShortsPreview({ VXID: vxId.value }),
    { transform: (data) => data.url },
);

const videoElement = useTemplateRef("videoElement");

const duration = ref<number | undefined>(0);
const startTime = ref<number | undefined>(0);
const endTime = ref<number | undefined>(0);

const shortDuration = computed(() => {
    if (startTime.value == undefined || endTime.value == undefined) return 0;
    return Math.ceil(endTime.value - startTime.value);
});

useEventListener(
    videoElement,
    "loadeddata",
    () => {
        duration.value = videoElement.value?.duration;
        startTime.value = 0;
        endTime.value = duration.value;
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
watch([duration, scrubberWidth], ([d, s]) => {
    if (!d || !s) return;
    zoom.value = s / d;
});

const toaster = useToast();
const confirmSubmit = ref(false);
async function submit() {
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
        navigateTo("/shorts");
        confirmSubmit.value = false;
    } catch (err) {
        toaster.create({
            title: "Failed to submit short",
            type: "error",
        });
    }
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
            <DesignButton icon="tabler:send" @click="confirmSubmit = true">
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
                    <DesignButton variant="primary" @click="submit">
                        {{ $t("shorts.generation.submitConfirmationSubmit") }}
                    </DesignButton>
                </div>
            </DesignDialog>
        </header>
        <template v-if="status == 'success'">
            <video
                ref="videoElement"
                :src="videoUrl"
                controls
                class="bg-surface-default aspect-video w-full shadow-xl"
            />
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
            <DesignSlider v-model="zoom" :min="0.1" :max="10" :step="0.01" />
        </template>
        <template v-if="status != 'success'">
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
