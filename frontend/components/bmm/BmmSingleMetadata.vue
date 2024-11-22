<script lang="ts" setup>
import {
    BccButton,
    BccModal,
    BccSelect,
    BccSpinner,
} from "@bcc-code/design-library-vue";
import {
    BmmEnvironment,
    BMMPermission,
    Language,
    Transcription,
} from "~/src/gen/api/v1/api_pb";

defineProps<{
    permissions: BMMPermission;
    environment: BmmEnvironment;
}>();

const form = defineModel<BMMSingleForm>({ required: true });

const albumId = computedProperty(form, "albumId");
const track = computedProperty(form, "track");
const language = computedProperty(form, "language");
const selectedEnvironment = computedProperty(form, "environment");

const emit = defineEmits<{
    set: [];
}>();

function checkForm() {
    if (!track.value) {
        alert("Please select a track");
        return;
    }

    emit("set");
}

const api = useAPI();
const transcription = ref<Transcription>();
const showTranscription = ref(false);
const transcriptionId = ref<string>();
const transcriptionLanguage = ref("nb");
const loadingTranscription = ref(false);

watch(transcriptionLanguage, () => {
    if (!transcriptionId.value) return;
    getTranscription(transcriptionId.value);
});

async function getTranscription(id: string) {
    loadingTranscription.value = true;
    showTranscription.value = true;
    try {
        transcription.value = await api.getBMMTranscription({
            language: transcriptionLanguage.value,
            bmmId: id,
        });
        transcriptionId.value = id;
    } catch (e) {
        showTranscription.value = false;
    } finally {
        loadingTranscription.value = false;
    }
}
</script>
<template>
    <form class="flex h-full flex-col gap-4 p-4" @submit.prevent="checkForm">
        <h3 class="text-heading-xl">BMM Upload</h3>

        <BccSelect
            v-if="permissions.integration"
            v-model="selectedEnvironment"
            :label="$t('Environment')"
        >
            <option value="prod">{{ $t("Production") }}</option>
            <option value="int">{{ $t("Integration") }}</option>
        </BccSelect>

        <AlbumSelector
            v-model="albumId"
            :permissions="permissions"
            :env="environment"
        />
        <BmmTrackSelector
            v-if="albumId"
            :key="albumId"
            label="Track"
            v-model="track"
            :album="albumId"
            :env="environment"
            @transcription="getTranscription"
        />
        <LanguageSelector
            v-model="language"
            :languages="permissions.languages"
            :env="environment"
        />

        <BccButton type="submit" class="mt-4">{{ $t("next") }}</BccButton>
    </form>
    <BccModal
        :open="showTranscription"
        @close="showTranscription = false"
        class="h-full min-w-[800px]"
        id="transcription-modal"
    >
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

        <template #header>
            <h2 class="text-heading-lg">Transcription</h2>
            <LanguageSelector
                :env="environment"
                v-model="transcriptionLanguage"
                :languages="permissions.languages"
            />
        </template>
    </BccModal>
</template>

<style>
#transcription-modal .bcc-modal-body {
    flex-grow: 1;
}
</style>
