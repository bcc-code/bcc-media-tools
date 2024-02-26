<script lang="ts" setup>
import { BccButton, BccInput } from "@bcc-code/design-library-vue";

defineProps<{
    languages: string[];
    albums: string[];
}>();

const form = defineModel<BMMSingleForm>({ required: true });

const albumId = computedProperty(form, "albumId");
const trackId = computedProperty(form, "trackId");
const language = computedProperty(form, "language");
const title = computedProperty(form, "title");

defineEmits<{
    set: [];
}>();
</script>
<template>
    <form class="flex flex-col gap-4 p-4" @submit.prevent="$emit('set')">
        <h3 class="text-lg font-bold">BMM Upload</h3>
        <AlbumSelector
            v-model="albumId"
            :label="$t('album')"
            :albums="albums"
        />
        <BmmTrackSelector
            v-if="albumId"
            :key="albumId"
            v-model="trackId"
            :album="albumId"
            :label="$t('track')"
        />
        <BccInput v-model="title" :label="$t('title')" required />
        <LanguageSelector v-model="language" :languages="languages" />
        <BccButton type="submit">{{ $t("next") }}</BccButton>
    </form>
</template>
