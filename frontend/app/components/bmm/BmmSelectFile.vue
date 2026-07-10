<script lang="ts" setup>
import type { BmmEnvironment } from "~~/src/gen/api/v1/api_pb";
import type { FileAndLanguage } from "~/utils/bmm";

const props = defineProps<{
    defaultLanguage: string;
    acceptMultiple: boolean;
    environment: BmmEnvironment;
}>();

const selectedFiles = defineModel<FileAndLanguage[]>({ required: true });
const selectedFilesComputed = computed({
    get() {
        return selectedFiles.value.map((f) => f.file);
    },
    set(value: File | File[]) {
        if (!Array.isArray(value)) value = [value];
        selectedFiles.value = value.map((f, i) => ({
            file: f,
            language: selectedFiles.value[i]?.language || props.defaultLanguage,
        }));
    },
});

const { me } = useMe();
const { isBmmAdmin } = usePermissions();
</script>

<template>
    <DesignFileUpload
        v-model="selectedFilesComputed"
        :multiple="props.acceptMultiple"
        accept="audio/mpeg,audio/wav"
        :label="$t('bmmUpload.addFiles')"
        :description="$t('bmmUpload.addFilesDescription')"
    >
        <template #file-trailing="{ index }">
            <BmmLanguageSelector
                v-if="selectedFiles[index] && me?.bmm"
                v-model="selectedFiles[index].language"
                :disabled="!isBmmAdmin"
                :languages="me.bmm.languages"
                :env="props.environment"
                label=""
            />
        </template>
    </DesignFileUpload>
</template>
