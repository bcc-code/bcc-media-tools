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
</script>

<template>
    <UFileUpload
        v-model="selectedFilesComputed"
        :multiple="props.acceptMultiple"
        accept="audio/mpeg"
        layout="list"
        :label="$t('bmmUpload.addFiles')"
        :description="$t('bmmUpload.addFilesDescription')"
    >
        <template #file-trailing="{ index }">
            <div class="ml-auto flex items-center gap-2">
                <BmmLanguageSelector
                    v-if="selectedFiles[index] && me?.bmm"
                    v-model="selectedFiles[index].language"
                    :disabled="!me.bmm.admin"
                    :languages="me.bmm.languages"
                    :env="props.environment"
                    label=""
                />
                <UButton
                    icon="heroicons:x-mark"
                    variant="ghost"
                    color="error"
                    square
                    @click="selectedFiles.splice(index, 1)"
                />
            </div>
        </template>
    </UFileUpload>
</template>
