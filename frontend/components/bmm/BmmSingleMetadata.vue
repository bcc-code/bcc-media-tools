<script lang="ts" setup>
import { BccAlert, BccButton, BccInput, BccSelect } from "@bcc-code/design-library-vue";
import { BmmEnvironment, BMMPermission } from "~/src/gen/api/v1/api_pb";

defineProps<{
    permissions: BMMPermission;
    environment: BmmEnvironment;
}>();

const form = defineModel<BMMSingleForm>({ required: true });

const albumId = computedProperty(form, "albumId");
const trackId = computedProperty(form, "trackId");
const language = computedProperty(form, "language");
const title = computedProperty(form, "title");
const selectedEnvironment = computedProperty(form, "environment");

const emit = defineEmits<{
    set: [];
}>();

function checkForm() {
    if (!trackId.value) {
        alert("Please select a track");
        return
    }

    emit('set')
}
</script>
<template>
    <form class="flex flex-col gap-4 p-4" @submit.prevent="checkForm">
        <h3 class="text-lg font-bold">BMM Upload</h3>

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
            v-model="trackId"
            :album="albumId"
            :env="environment"
        />
        <div class="flex flex-col gap-2 border-2 border-slate-950 p-4">
            <LanguageSelector
                v-model="language"
                :languages="permissions.languages"
                :env="environment"
            />
            <BccInput v-model="title" :label="$t('title')" required />
            <bcc-alert context="info" :icon="true">The title will only be updated for the selected language in BMM if there is no title already present</bcc-alert>
        </div>
        <BccButton type="submit" >{{ $t("next") }}</BccButton>
    </form>
</template>
