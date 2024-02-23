<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";

const selectedFile = defineModel<File | null>({ required: true });

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
};

const fileInput = ref<HTMLInputElement>(null!);

const selectFile = (event: any) => {
    selectedFile.value = event.target?.files[0];
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
        <div v-if="selectedFile" class="m-auto text-center text-lg">
            <p>{{ selectedFile.name }}</p>
            <BccButton @click.stop="selectedFile = null" variant="secondary">
                Clear
            </BccButton>
        </div>
        <template v-else>
            <p class="m-auto text-lg">{{ $t("dragAndDropFileHere") }}</p>
        </template>
        <input
            ref="fileInput"
            type="file"
            class="hidden"
            @change="selectFile"
        />
    </div>
</template>
