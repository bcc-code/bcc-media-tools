<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";

const props = defineProps<{
    endpoint: string;
}>();

const selectedFile = defineModel<File | null>({ required: true });
const uploadPercentage = ref(0);

watch(selectedFile, () => {
    uploadPercentage.value = 0;
});

const uploadFile = () => {
    if (!selectedFile.value) return;

    const formData = new FormData();
    formData.append("file", selectedFile.value);

    const xhr = new XMLHttpRequest();
    xhr.open("post", props.endpoint, true);
    xhr.upload.onprogress = function (ev) {
        // Upload progress here
        uploadPercentage.value = Math.floor((ev.loaded / ev.total) * 1000) / 10;
    };
    xhr.onreadystatechange = function () {
        if (xhr.readyState === 4) {
            // Uploaded
            console.log("Uploaded");
        }
    };
    xhr.send(formData);
};
</script>

<template>
    <div class="flex w-full flex-col gap-4">
        <BccButton @click="uploadFile" :disabled="!selectedFile">
            Upload
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
