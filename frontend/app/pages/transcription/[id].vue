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
const routeId = route.params.id;

const {
    segments,
    loading,
    error,
    saveState,
    savedAt,
    submitting,
    load,
    reset,
    submit,
} = useTranscriptionDraft(routeId);

const fileName = computed(() => `transcription-${routeId}`);

const video = ref<string>();
const videoelement = ref<HTMLVideoElement>();

const segmentelements = ref<{
    [key: number]: ComponentPublicInstance;
}>({});

const { t } = useI18n();
const toaster = useToast();

// i18n has no `datetimeFormats` configured.
const savedAtLabel = computed(() =>
    savedAt.value
        ? savedAt.value.toLocaleTimeString(undefined, {
              hour: "2-digit",
              minute: "2-digit",
          })
        : null,
);

const handleReset = async () => {
    if (await reset()) {
        toaster.create({
            title: t("transcription.resetSuccess"),
            type: "success",
        });
    }
};

const showSubmitConfirmationModal = ref(false);
const submitToMediabanken = async () => {
    if (await submit()) {
        toaster.create({
            title: t("transcription.submitSuccess"),
            type: "success",
        });
        navigateTo("/transcription");
    } else {
        toaster.create({
            title: t("transcription.submitError"),
            type: "error",
        });
    }
};

onMounted(async () => {
    try {
        video.value = (
            await api.getTranscriptionPreview({ VXID: routeId })
        ).url;
    } catch (e: unknown) {
        error.value = (e as { message?: string })?.message ?? "Unknown error";
        loading.value = false;
        return;
    }

    await load();
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

            if (index === null) return;
            if (index === prevIndex) return;

            focusedSegment.value = segments.value[index];

            // Virtualised: a segment outside the window has no element.
            const segmentElement = segmentelements.value[index]?.$el as
                HTMLDivElement | undefined;
            segmentElement?.scrollIntoView({
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
    if (seekOnFocus.value) {
        if (el.fastSeek) {
            el.fastSeek(word.start);
        } else {
            el.currentTime = word.start;
        }
    }
};

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
            class="border-border-1 bg-surface-default flex items-center justify-between gap-4 border-b px-6 py-3"
        >
            <div class="flex flex-col">
                <div class="flex items-center gap-3">
                    <p v-if="saveState === 'error'" class="text-semantic-error">
                        {{ $t("transcription.saveFailed") }}
                    </p>
                    <p v-else-if="savedAtLabel">
                        {{
                            $t("transcription.changesSavedLocallyAt", {
                                time: savedAtLabel,
                            })
                        }}
                    </p>
                    <p v-else>{{ $t("transcription.changesSavedLocally") }}</p>
                    <button
                        class="-m-3 p-3 text-neutral-500 underline"
                        @click="handleReset"
                    >
                        {{ $t("transcription.reset") }}
                    </button>
                </div>
            </div>
            <div class="flex items-center gap-4">
                <DesignSwitch
                    v-model="previewSubtitles"
                    :label="$t('transcription.previewSubtitles')"
                />
                <DesignSwitch
                    v-model="seekOnFocus"
                    :label="$t('transcription.seekOnFocus')"
                />
                <DesignSwitch
                    v-model="deleteMode"
                    :label="$t('transcription.deleteMode')"
                />
                <TranscriptionDownloader
                    :segments="segments"
                    :filename="fileName"
                />
                <DesignButton @click="showSubmitConfirmationModal = true">
                    {{ $t("transcription.save") }}
                </DesignButton>
                <button
                    class="-mx-3 aspect-square p-3"
                    @click="showManual = true"
                >
                    <Icon name="tabler:help-circle" class="text-xl" />
                </button>
            </div>
        </div>
        <DesignBanner
            v-if="saveState === 'error'"
            variant="error"
            icon="tabler:alert-triangle"
            class="mx-6 mt-3"
        >
            {{ $t("transcription.saveFailedDescription") }}
        </DesignBanner>
        <div
            v-bind="splitterApi.getRootProps()"
            class="flex bg-neutral-100 dark:bg-neutral-950"
        >
            <div
                v-bind="splitterApi.getPanelProps({ id: 'left' })"
                class="bg-surface-default border-border-1 flex flex-col border-r"
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
                    {{ error }}
                </div>
                <TranscriptionEditor
                    class="ml-auto w-full max-w-7xl overflow-auto"
                    v-if="segments.length && !loading"
                    v-model="segments"
                    v-model:segmentelements="segmentelements"
                    :focused-segment="focusedSegment"
                    @word-focus="handleWordFocus"
                />
            </div>
            <div class="flex h-full items-center px-1">
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
                <Icon
                    v-if="loading && !video"
                    name="svg-spinners:bars-rotate-fade"
                    class="m-auto text-2xl"
                />
                <div class="relative mx-auto p-4">
                    <template v-if="video">
                        <video
                            ref="videoelement"
                            :src="video"
                            controls
                            class="bg-surface-default shadow-xl"
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
        <DesignDialog
            v-model:open="showSubmitConfirmationModal"
            :title="$t('transcription.submitConfirmationTitle')"
            :description="$t('transcription.submitConfirmationMessage')"
        >
            <div class="flex w-full justify-end gap-2">
                <DesignButton
                    variant="tertiary"
                    @click="showSubmitConfirmationModal = false"
                >
                    {{ $t("transcription.submitConfirmationCancel") }}
                </DesignButton>
                <DesignButton
                    variant="primary"
                    :loading="submitting"
                    :disabled="submitting"
                    @click="submitToMediabanken"
                >
                    {{ $t("transcription.submitConfirmationSubmit") }}
                </DesignButton>
            </div>
        </DesignDialog>
    </div>
</template>

<style>
[data-scope="splitter"][data-part="resize-trigger"] {
    height: calc(var(--spacing) * 16);
    width: calc(var(--spacing) * 2);
    border-radius: calc(infinity * 1px);
    background-color: var(--color-text-hint);
    transition: background-color 150ms;

    &:hover {
        background-color: var(--color-text-muted);
    }
}
</style>
