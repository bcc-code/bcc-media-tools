<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";
import type { FileAndLanguage } from "~/utils/bmm";

const props = defineProps<{
    endpoint: string;
    files: FileAndLanguage[];
    metadata: { [key: string]: readonly string[] };
}>();

const emit = defineEmits<{
    uploaded: [];
}>();

const selectedFiles = defineModel<FileAndLanguage[]>({ required: true });
const uploadPercentageFiles = ref<{ [key: string]: number }>({});
const uploading = ref(false);
const uploadPercentage = ref(0);

watch(uploadPercentageFiles, () => {
    uploadPercentage.value = Object.values(uploadPercentageFiles.value).reduce((a, b) => a + b, 0) / Object.keys(uploadPercentageFiles.value).length;
}, {deep: true});

watch(selectedFiles, () => {
    uploadPercentage.value = 0;
    uploadPercentageFiles.value = {};
    uploading.value = false;
});

const abort = ref<() => void>();

const uploadFile = () => {
    for (const selectedFile of selectedFiles.value || []) {

        if (!selectedFile.file) return;
        uploading.value = true;

        const formData = new FormData();
        formData.append("file", selectedFile.file);
        formData.append("file_language", selectedFile.language);
        if (props.metadata) {
            for (const [key, values] of Object.entries(props.metadata)) {
                for (const value of values) {
                    formData.append(key, value);
                }
            }
        }

        const xhr = new XMLHttpRequest();
        xhr.open("post", props.endpoint, true);
        xhr.upload.onprogress = function (ev) {
            // Upload progress here
            uploadPercentageFiles.value[selectedFile.file.name] = Math.floor((ev.loaded / ev.total) * 1000) / 10;
        };
        xhr.onreadystatechange = function () {
            if (xhr.readyState === 4) {
                // Uploaded
                emit("uploaded");
            }
        };
        xhr.send(formData);
        abort.value = () => {
            xhr.abort();
            uploading.value = false;
        };
    }
};
</script>

<template>
    <div class="flex w-full flex-col gap-4">
        <BccButton
            @click="uploadFile"
            v-if="!uploading"
            :disabled="selectedFiles.length < 1"
        >
            Upload
        </BccButton>
        <BccButton
            @click="abort"
            v-else
            variant="secondary"
            :disabled="uploadPercentage >= 100"
        >
            Cancel
        </BccButton>
        <div class="flex justify-between">
            <div class="grow">
                <div
                    class="flex h-8 bg-green-600"
                    :class="
                        uploadPercentage !== 100 ? 'rounded-lg' : 'rounded-l-lg'
                    "
                    :style="{ width: `${uploadPercentage}%` }"
                ></div>
            </div>
            <span
                class="m-auto bg-slate-600 px-2 py-1 font-bold text-white"
                :class="
                    uploadPercentage !== 100 ? 'rounded-lg' : 'rounded-r-lg'
                "
            >
                {{
                    uploadPercentage !== 100
                        ? uploadPercentage.toFixed(1)
                        : 100
                }}%
            </span>
        </div>
    </div>
</template>
