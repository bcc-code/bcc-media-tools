<template>
    <div
        class="mx-auto flex h-screen max-w-screen-md flex-col gap-4 rounded-lg bg-stone-300 p-4 text-black"
    >
        <form
            class="flex flex-col gap-4 p-4"
            @submit.prevent="metadataIsSet = true"
        >
            <h3 class="text-lg font-bold">BMM Upload</h3>
            <BccInput :label="$t('originalTitle')" required />
            <BccSelect :label="$t('language')" required>
                <option disabled value="">Select an option...</option>
                <option v-for="l in languages" :value="l.key">
                    {{ l.value }}
                </option>
            </BccSelect>
            <BccButton type="submit">Next</BccButton>
        </form>
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
import { BccButton, BccInput, BccSelect } from "@bcc-code/design-library-vue";

const metadataIsSet = ref(false);

const selectedFile = ref<File | null>(null);

const languages = [
    {
        key: "en",
        value: "English",
    },
    {
        key: "es",
        value: "Spanish",
    },
    {
        key: "fr",
        value: "French",
    },
];
</script>
