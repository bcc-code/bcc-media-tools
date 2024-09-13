<script lang="ts" setup>
import { BccButton, BccSelect } from "@bcc-code/design-library-vue";
import { BmmEnvironment, BMMPermission } from "~/src/gen/api/v1/api_pb";

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
</script>
<template>
    <form class="flex flex-col gap-4 p-4" @submit.prevent="checkForm">
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
        />
        <LanguageSelector
            v-model="language"
            :languages="permissions.languages"
            :env="environment"
        />

        <BccButton type="submit" class="mt-4">{{ $t("next") }}</BccButton>
    </form>
</template>
