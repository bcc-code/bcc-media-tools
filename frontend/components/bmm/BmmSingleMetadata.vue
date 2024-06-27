<script lang="ts" setup>
import { BccButton, BccInput, BccSelect } from "@bcc-code/design-library-vue";
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

defineEmits<{
    set: [];
}>();
</script>
<template>
    <form class="flex flex-col gap-4 p-4" @submit.prevent="$emit('set')">
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
        <BccInput v-model="title" :label="$t('title')" required />
        <LanguageSelector
            v-model="language"
            :languages="permissions.languages"
            :env="environment"
        />
        <BccButton type="submit">{{ $t("next") }}</BccButton>
    </form>
</template>
