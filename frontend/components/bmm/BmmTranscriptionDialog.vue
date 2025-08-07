<script setup lang="ts">
import type { BmmEnvironment, BMMTrack } from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    env: BmmEnvironment;
}>();

const api = useAPI();

const track = defineModel<BMMTrack>("track");
const showTranscription = computed({
    get: () => !!track.value,
    set: (v) => {
        if (!v) {
            resetTranscription();
        }
    },
});
const transcriptionLanguages = ref<string[]>();
const transcriptionLanguage = ref<string>();

const analytics = useAnalytics();
watch(showTranscription, (s) => {
    if (!s) return;

    analytics.page({
        id: "bmm_transcription",
        title: "bmm transcription",
    });
});

watch(track, (t) => {
    if (!t) return;

    // Set available transcription languages, and default to first
    transcriptionLanguages.value = t.transcriptions?.Languages.map(
        (l) => l.code,
    );
    transcriptionLanguage.value = transcriptionLanguages.value?.at(0) || "nb";

    analytics.track("transcription_loaded", {
        language: transcriptionLanguage.value,
        trackId: t.id,
    });
});

const { data: transcription, status } = useLazyAsyncData(
    () =>
        `${props.env}:${track.value?.id}:${transcriptionLanguage.value}:transcription`,
    async () => {
        if (!track.value) return;
        return api
            .getBMMTranscription({
                language: transcriptionLanguage.value,
                bmmId: track.value.id,
                environment: props.env,
            })
            .catch((e) => {
                resetTranscription();
                throw e;
            });
    },
);

function resetTranscription() {
    track.value = undefined;
    transcription.value = undefined;
    transcriptionLanguage.value = undefined;
    transcriptionLanguages.value = undefined;
}

const { t } = useI18n();
const toast = useToast();
function copyToClipboard() {
    if (!transcription.value) return;
    const text = transcription.value.segments.map((s) => s.text).join(" ");
    navigator.clipboard.writeText(text);
    toast.add({
        icon: "heroicons:check",
        title: t("bmmUpload.copiedToClipboard"),
        color: "success",
    });
}
</script>

<template>
    <UModal
        class="h-full w-full max-w-[800px]"
        dismissible
        v-model:open="showTranscription"
    >
        <template #header>
            <div class="flex w-full items-start justify-between gap-4">
                <div>
                    <h2 class="mb-2 text-xl font-bold">
                        {{ $t("bmmUpload.transcription") }}
                    </h2>
                    <BmmLanguageSelector
                        v-if="transcriptionLanguages?.length"
                        v-model="transcriptionLanguage"
                        :env="env"
                        :languages="transcriptionLanguages"
                    />
                </div>
                <UButton type="button" @click="copyToClipboard">
                    {{ $t("bmmUpload.copyToClipboard") }}
                </UButton>
            </div>
        </template>

        <template #body>
            <template v-if="transcription && status == 'success'">
                <p
                    v-for="segment in transcription.segments"
                    class="leading-relaxed"
                >
                    {{ segment.text }}
                </p>
            </template>
            <div v-else-if="status == 'pending'">
                <Icon
                    name="svg-spinners:bars-rotate-fade"
                    class="absolute top-1/2 left-1/2 size-8"
                />
            </div>
        </template>
    </UModal>
</template>
