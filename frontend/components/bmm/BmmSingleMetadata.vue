<script lang="ts" setup>
import {BccButton, BccInput, BccSelect} from "@bcc-code/design-library-vue";
import {BmmEnvironment} from "~/src/gen/api/v1/api_pb";

defineProps<{
    languages: string[];
    albums: string[];
}>();

const form = defineModel<BMMSingleForm>({ required: true });

const albumId = computedProperty(form, "albumId");
const trackId = computedProperty(form, "trackId");
const language = computedProperty(form, "language");
const title = computedProperty(form, "title");
const selectedEnvironment = ref("prod");
const env = computed(() => selectedEnvironment.value === "int"? BmmEnvironment.Integration : BmmEnvironment.Production);

defineEmits<{
    set: [];
}>();
</script>
<template>
    <form class="flex flex-col gap-4 p-4" @submit.prevent="$emit('set')">
        <h3 class="text-lg font-bold">BMM Upload</h3>

      <BccSelect v-model="selectedEnvironment" :label="$t('Environment')">
        <option value="prod">{{ $t("Production") }}</option>
        <option value="int">{{ $t("Integration") }}</option>
      </BccSelect>

        <AlbumSelector
            v-model="albumId"
            :users-albums="albums"
            :env="env"
        />
        <BmmTrackSelector
            v-if="albumId"
            :key="albumId"
            v-model="trackId"
            :album="albumId"
            :env="env"
        />
        <BccInput v-model="title" :label="$t('title')" required />
        <LanguageSelector v-model="language" :languages="languages" />
        <BccButton type="submit">{{ $t("next") }}</BccButton>
    </form>
</template>
