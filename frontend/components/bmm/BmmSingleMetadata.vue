<script lang="ts" setup>
import {BccButton, BccInput, BccSelect} from "@bcc-code/design-library-vue";
import {BmmEnvironment, BMMPermission} from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    permissions: BMMPermission;
}>();

const form = defineModel<BMMSingleForm>({ required: true });

const albumId = computedProperty(form, "albumId");
const trackId = computedProperty(form, "trackId");
const language = computedProperty(form, "language");
const title = computedProperty(form, "title");
const selectedEnvironment = ref("prod");
const env = computed(() => selectedEnvironment.value === "int" && props.permissions.integration ? BmmEnvironment.Integration : BmmEnvironment.Production);

defineEmits<{
    set: [];
}>();
</script>
<template>
    <form class="flex flex-col gap-4 p-4" @submit.prevent="$emit('set')">
        <h3 class="text-lg font-bold">BMM Upload</h3>

      <BccSelect v-if="permissions.integration" v-model="selectedEnvironment" :label="$t('Environment')">
        <option value="prod">{{ $t("Production") }}</option>
        <option value="int">{{ $t("Integration") }}</option>
      </BccSelect>

        <AlbumSelector
            v-model="albumId"
            :permissions="permissions"
            :env="env"
        />
        <BmmTrackSelector
            v-if="albumId"
            :key="albumId"
            label="Track"
            v-model="trackId"
            :album="albumId"
            :env="env"
        />
        <BccInput v-model="title" :label="$t('title')" required />
        <LanguageSelector v-model="language" :languages="permissions.languages" :env="env" />
        <BccButton type="submit">{{ $t("next") }}</BccButton>
    </form>
</template>
