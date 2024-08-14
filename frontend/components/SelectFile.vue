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

    if (!files) return;

    for (const file of files) {
        if (!file.type.startsWith("audio/")) {
            continue;
        }

        selectedFiles.value.push({ file, language: props.defaultLanguage });

        if (!props.acceptMultiple) {
            break;
        }
    }
};

const fileInput = ref<HTMLInputElement>(null!);

const selectFile = ( event: Event ) => {
    const target = event.target as HTMLInputElement;
    for (const file of target.files??[]) {
        selectedFiles.value.push({
            file: file as File,
            language: props.defaultLanguage,
        });
    }
};

const props = defineProps<{
    defaultLanguage: string;
    acceptMultiple: boolean;
}>();

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
            accept="audio/*"
            :multiple="props.acceptMultiple ?? null"
            @change="selectFile"
        />
    </div>
</template>
