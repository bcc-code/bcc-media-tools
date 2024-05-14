<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";

const props = defineProps<{
    endpoint: string;
    metadata?: { [key: string]: readonly string[] };
}>();

const emit = defineEmits<{
    uploaded: [];
}>();

const selectedFile = defineModel<File | null>({ required: true });
const uploadPercentage = ref(0);
const uploading = ref(false);

watch(selectedFile, () => {
    uploadPercentage.value = 0;
    uploading.value = false;
});

const abort = ref<() => void>();

const uploadFile = () => {
    if (!selectedFile.value) return;
    uploading.value = true;

    const formData = new FormData();
    formData.append("file", selectedFile.value);
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
        uploadPercentage.value = Math.floor((ev.loaded / ev.total) * 1000) / 10;
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
};
</script>

<template>
    <div class="flex w-full flex-col gap-4">
        <BccButton
            @click="uploadFile"
            v-if="!uploading"
            :disabled="!selectedFile"
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
