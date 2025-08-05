<script lang="ts" setup>
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
    <form class="flex h-full flex-col gap-4 p-4" @submit.prevent="checkForm">
        <h3 class="text-2xl font-bold">BMM Upload</h3>

        <UFormField v-if="permissions.integration" :label="$t('Environment')">
            <USelectMenu
                v-model="selectedEnvironment"
                value-key="value"
                label-key="label"
                :items="[
                    { label: $t('Integration'), value: 'int' },
                    { label: $t('Production'), value: 'prod' },
                ]"
            />
        </UFormField>

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

        <UButton type="submit" class="mt-4" block size="lg">
            {{ $t("next") }}
        </UButton>
    </form>
</template>
