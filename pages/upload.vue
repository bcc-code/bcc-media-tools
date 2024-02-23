<template>
    <div class="mx-auto flex h-screen max-w-screen-md flex-col gap-4 p-8">
        <div
            class="bg-gray mx-auto flex h-48 w-full cursor-pointer rounded-lg border-2 text-center transition hover:bg-slate-700"
            @click="fileInput?.click()"
            :class="[isDragOver ? 'border-green-500' : 'border-gray-300']"
            @dragenter.prevent="dragEnter"
            @dragover.prevent
            @dragleave.prevent="dragLeave"
            @drop.prevent="handleDrop"
        >
            <div v-if="selectedFile" class="m-auto text-center text-lg">
                <p>{{ selectedFile.name }}</p>
                <button
                    class="button bg-slate-600"
                    @click.stop="selectedFile = null"
                >
                    Clear
                </button>
            </div>
            <template v-else>
                <p class="m-auto text-lg">Drag and drop file here</p>
            </template>
            <input
                ref="fileInput"
                type="file"
                class="hidden"
                @change="selectFile"
            />
        </div>
        <div class="flex w-full flex-col gap-4">
            <button
                class="button mx-auto bg-slate-700"
                :class="{ 'cursor-not-allowed': !selectedFile }"
                @click="uploadFile"
                :disabled="!selectedFile"
            >
                Upload
            </button>
            <div class="flex justify-between">
                <div class="grow">
                    <div
                        class="flex h-8 rounded bg-green-500"
                        :style="{ width: `${uploadPercentage}%` }"
                    ></div>
                </div>
                <span class="m-auto bg-slate-800 px-4">
                    {{ uploadPercentage }}%
                </span>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
const isDragOver = ref(false);
const dragEnter = () => {
    isDragOver.value = true;
};

const dragLeave = () => {
    isDragOver.value = false;
};

const handleDrop = (event: DragEvent) => {
    isDragOver.value = false;
    const files = event.dataTransfer?.files;
    selectedFile.value = files?.[0] ?? null;
    uploadPercentage.value = 0; // Reset progress bar
};

const fileInput = ref<HTMLInputElement>(null!);
const selectedFile = ref<File | null>(null);
const uploadPercentage = ref(0);

const selectFile = (event: any) => {
    selectedFile.value = event.target?.files[0];
    uploadPercentage.value = 0; // Reset progress bar
};

const uploadFile = () => {
    if (!selectedFile.value) return;

    const formData = new FormData();
    formData.append("file", selectedFile.value);

    const xhr = new XMLHttpRequest();
    xhr.open("post", "/api/files/upload", true);
    xhr.upload.onprogress = function (ev) {
        // Upload progress here
        uploadPercentage.value = Math.floor((ev.loaded / ev.total) * 100);
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

<style scoped>
.button {
    @apply rounded px-2 py-1;
}
</style>
