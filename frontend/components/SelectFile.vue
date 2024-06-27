<script lang="ts" setup>
import type { FileAndLanguage } from "~/utils/bmm";

const selectedFiles = defineModel<FileAndLanguage[]>({ required: true });

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
    const file = files?.[0];
    if (file) selectedFiles.value.push({ file, language: "nb" });
};

const fileInput = ref<HTMLInputElement>(null!);

const selectFile = (
    event: Event & { target: EventTarget & HTMLInputElement },
) => {
    selectedFiles.value.push({
        file: event.target?.files?.[0],
        language: "nb",
    });
};
</script>

<template>
    <div
        class="bg-gray mx-auto flex h-48 w-full cursor-pointer rounded-lg border-2 bg-slate-800 text-center text-white transition hover:bg-slate-700"
        @click="fileInput?.click()"
        :class="[isDragOver ? 'border-green-500' : 'border-slate-700']"
        @dragenter.prevent="dragEnter"
        @dragover.prevent
        @dragleave.prevent="dragLeave"
        @drop.prevent="handleDrop"
    >
        <p class="m-auto text-lg">{{ $t("dragAndDropFileHere") }}</p>
        <input
            ref="fileInput"
            type="file"
            class="hidden"
            @change="selectFile"
        />
    </div>
</template>
