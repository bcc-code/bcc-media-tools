<script lang="ts" setup>
import { BmmEnvironment } from "~~/src/gen/api/v1/api_pb";
import type { BMMPermission } from "~~/src/gen/api/v1/api_pb";

defineProps<{
    permissions: BMMPermission;
    environment: BmmEnvironment;
}>();

const form = defineModel<BMMSingleForm>({ required: true });

const albumId = computedProperty(form, "albumId");
const track = computedProperty(form, "track");
const language = computedProperty(form, "language");
const selectedEnvironment = computedProperty(form, "environment");
const contentType = computedProperty(form, "contentType");

const emit = defineEmits<{
    set: [];
}>();

const { t } = useI18n();
function checkForm() {
    if (!track.value) {
        alert(t("bmmUpload.trackRequiredAlert"));
        return;
    }

    emit("set");
}
</script>
<template>
    <form class="flex h-full flex-col gap-4 p-4" @submit.prevent="checkForm">
        <h3 class="text-heading-3 text-text-default">
            {{ $t("bmmUpload.title") }}
        </h3>

        <div v-if="permissions.integration" class="flex flex-col gap-1">
            <label class="text-body-3 text-text-muted block">
                {{ $t("bmmUpload.environment") }}
            </label>
            <DesignSelect
                v-model="selectedEnvironment"
                :items="[
                    { label: 'Integration', value: 'int' },
                    { label: 'Production', value: 'prod' },
                ]"
            />
        </div>
        <BmmAlbumSelector
            v-model="albumId"
            v-model:content-type="contentType"
            :permissions="permissions"
            :env="environment"
        />
        <BmmTrackSelector
            v-if="albumId"
            :key="albumId"
            :label="$t('bmmUpload.track')"
            v-model="track"
            :album="albumId"
            :env="environment"
        />
        <BmmLanguageSelector
            v-model="language"
            :languages="permissions.languages"
            :env="environment"
        />

        <DesignButton type="submit" size="large" class="mt-4 w-full">
            {{ $t("bmmUpload.next") }}
        </DesignButton>
    </form>
</template>
