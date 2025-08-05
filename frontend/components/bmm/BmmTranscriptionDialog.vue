<script setup lang="ts">
import type {
    BmmEnvironment,
    BMMTrack,
    Transcription,
} from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    env: BmmEnvironment;
}>();

const track = defineModel<BMMTrack>("track");

const api = useAPI();

const transcription = ref<Transcription>();
const showTranscription = ref(false);
const transcriptionLanguages = ref<string[]>();
const transcriptionLanguage = ref<string>();
const loadingTranscription = ref(false);

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
watch(transcriptionLanguage, (t) => {
    if (!t) return;
    getTranscription();
});

async function getTranscription() {
    if (!track.value) return;

    loadingTranscription.value = true;
    showTranscription.value = true;

    try {
        transcription.value = await api.getBMMTranscription({
            language: transcriptionLanguage.value,
            bmmId: track.value.id,
            environment: props.env,
        });
    } catch (e) {
        resetTranscription();
    } finally {
        loadingTranscription.value = false;
    }
}

function resetTranscription() {
    showTranscription.value = false;
    track.value = undefined;
    transcription.value = undefined;
    transcriptionLanguage.value = undefined;
    transcriptionLanguages.value = undefined;
}

const toast = useToast();
function copyToClipboard() {
    if (!transcription.value) return;
    const text = transcription.value.segments.map((s) => s.text).join(" ");
    navigator.clipboard.writeText(text);
    toast.add({
        title: "Copied to clipboard",
    });
}
</script>

<template>
    <UModal
        class="h-full w-full max-w-[800px]"
        dismissible
        title="Transcript"
        v-model:open="showTranscription"
        @after:leave="resetTranscription"
    >
        <template #header="{ close }">
            <div class="flex w-full items-start justify-between gap-4">
                <div>
                    <h2 class="mb-2 text-xl font-bold">Transcript</h2>
                    <LanguageSelector
                        v-if="transcriptionLanguages?.length"
                        v-model="transcriptionLanguage"
                        :env="env"
                        :languages="transcriptionLanguages"
                    />
                </div>
                <UButton
                    type="button"
                    @click="
                        () => {
                            copyToClipboard();
                            close();
                        }
                    "
                >
                    Copy to clipboard
                </UButton>
            </div>
        </template>

        <template #body>
            <template v-if="transcription && !loadingTranscription">
                <p
                    v-for="segment in transcription.segments"
                    class="leading-relaxed"
                >
                    {{ segment.text }}
                </p>
            </template>
            <div v-else-if="loadingTranscription">
                <Icon
                    name="svg-spinners:bars-rotate-fade"
                    class="absolute top-1/2 left-1/2 size-8"
                />
            </div>
        </template>
    </UModal>
</template>
