<script setup lang="ts">
import { BccButton, BccModal, BccSpinner } from "@bcc-code/design-library-vue";
import type {
    BmmEnvironment,
    BMMTrack,
    Transcription,
} from "~/src/gen/api/v1/api_pb";
import { analytics } from "~/utils/analytics";

analytics.page({
    id: "bmm_transcription",
    title: "bmm transcription",
});

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
    })

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

const { $toast } = useNuxtApp();
function copyToClipboard() {
    if (!transcription.value) return;
    const text = transcription.value.segments.map((s) => s.text).join(" ");
    navigator.clipboard.writeText(text);
    $toast.success("Copied to clipboard");
}
</script>

<template>
    <BccModal
        id="bmm-transcription-modal"
        class="h-full w-full max-w-[800px]"
        :open="showTranscription"
        @close="resetTranscription"
    >
        <template #header>
            <div class="flex justify-between gap-4">
                <div>
                    <h2 class="text-heading-xl">Transcript</h2>
                    <LanguageSelector
                        v-if="transcriptionLanguages?.length"
                        v-model="transcriptionLanguage"
                        :env="env"
                        :languages="transcriptionLanguages"
                    />
                </div>
            </div>
        </template>

        <template v-if="transcription && !loadingTranscription">
            <p
                v-for="segment in transcription.segments"
                class="leading-relaxed"
            >
                {{ segment.text }}
            </p>
        </template>
        <div v-else-if="loadingTranscription">
            <BccSpinner size="sm" class="absolute left-1/2 top-1/2" />
        </div>

        <template #secondaryAction>
            <BccButton
                variant="secondary"
                type="button"
                @click="copyToClipboard"
            >
                Copy to clipboard
            </BccButton>
        </template>
    </BccModal>
</template>

<style>
#bmm-transcription-modal .bcc-modal-body {
    flex-grow: 1;
}
</style>
