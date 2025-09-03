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

const { data: videoUrl, status } = useAsyncData(
    () => `preview:${vxId.value}`,
    () => api.getPreview({ VXID: vxId.value }),
    { transform: (data) => data.url },
);

const videoElement = useTemplateRef("videoElement");

const duration = ref<number | undefined>(0);
const startTime = ref<number | undefined>(0);
const endTime = ref<number | undefined>(0);

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

const currentTime = ref(0);
useEventListener(videoElement, "timeupdate", () => {
    if (!videoElement.value) return;
    currentTime.value = videoElement.value.currentTime;
});

function previewShort() {
    if (
        !videoElement.value ||
        startTime.value == undefined ||
        endTime.value == undefined
    )
        return;

    videoElement.value.currentTime = startTime.value;
    videoElement.value.play();

    const timeoutDuration = (endTime.value - startTime.value) * 1000;
    const timeout = setTimeout(() => {
        if (videoElement.value) {
            videoElement.value.pause();
        }
        clearTimeout(timeout);
    }, timeoutDuration);
}

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

const confirmSubmit = ref(false);
function submit() {
    confirmSubmit.value = false;
}
</script>

<template>
    <div class="mx-auto flex w-full max-w-7xl flex-col gap-4 p-8">
        <header class="mb-4 flex items-center justify-between">
            <div>
                <h1 class="text-2xl font-bold">Shorts generation</h1>
                <p class="text-muted text-sm">
                    A simple tool to generate simple shorts.
                </p>
            </div>
            <UButton @click="confirmSubmit = true">
                <UIcon name="tabler:send" class="text-dimmed" />
                Submit and Generate
            </UButton>
            <UModal
                v-model:open="confirmSubmit"
                title="Submit and Generate Short"
                description="Are you sure you want to submit?"
            >
                <template #footer>
                    <div class="flex w-full justify-end gap-2">
                        <UButton @click="confirmSubmit = false" variant="ghost">
                            Cancel
                        </UButton>
                        <UButton @click="submit">Submit</UButton>
                    </div>
                </template>
            </UModal>
        </header>
        <template v-if="status == 'success'">
            <video
                ref="videoElement"
                :src="videoUrl"
                controls
                class="bg-default aspect-video w-full shadow-xl"
            />
            <div class="flex items-center gap-2">
                <p
                    v-if="startTime != undefined && endTime != undefined"
                    class="tabular-nums"
                >
                    {{ formatTime(startTime) }} - {{ formatTime(endTime) }}
                </p>
                <UButton
                    class="ml-auto"
                    variant="soft"
                    @click="startTime = currentTime"
                >
                    Set start point
                </UButton>
                <UButton variant="soft" @click="endTime = currentTime">
                    Set end point
                </UButton>
                <UButton variant="soft" @click="previewShort">
                    Preview short
                </UButton>
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
                v-model:start="startTime"
                v-model:end="endTime"
            />
            <USlider v-model="zoom" :min="0.1" :max="10" :step="0.01" />
        </template>
        <template v-if="status != 'success'">
            <USkeleton class="aspect-video w-full" />
            <div class="flex items-center gap-2">
                <USkeleton class="h-6 w-48" />
                <USkeleton class="ml-auto h-8 w-28" />
                <USkeleton class="h-8 w-28" />
                <USkeleton class="h-8 w-28" />
            </div>
            <USkeleton class="h-32 w-full" />
            <USkeleton class="h-2 w-full" />
        </template>

        <UModal v-model:open="showManual">
            <template #body>
                <img :src="manualGif" />
            </template>
        </UModal>
    </div>
</template>
