<template>
    <div
        class="mx-auto flex h-screen max-w-screen-md flex-col gap-4 rounded-lg bg-stone-300 p-4 text-black"
    >
        <BmmSingleMetadata v-model="form" @submit="metadataIsSet = true" />
        <div
            class="flex flex-col gap-4 p-4 transition"
            :class="[
                {
                    'pointer-events-none opacity-50': !metadataIsSet,
                },
            ]"
        >
            <h3 class="text-lg font-bold">Upload File</h3>
            <SelectFile v-model="selectedFile" />
            <FileUploader
                v-model="selectedFile"
                endpoint="/api/files/upload/bmm"
            />
        </div>
    </div>
</template>

<script lang="ts" setup>
const form = ref<BMMSingleForm>({
    originalTitle: "",
});

const metadataIsSet = ref(false);

const selectedFile = ref<File | null>(null);
</script>
