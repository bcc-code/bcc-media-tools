<script lang="ts" setup>
const form = ref<BMMSingleForm>({
    title: "",
});

const metadataIsSet = ref(false);

const selectedFile = ref<File | null>(null);

const me = ref<{
    languages: string[];
    albums: string[];
}>();

onMounted(async () => {
    me.value = await $fetch("/api/bmm/me", {
        method: "GET",
    });
});
</script>

<template>
    <div
        class="mx-auto flex min-h-screen max-w-screen-md flex-col gap-4 rounded-lg bg-stone-300 p-4 text-black"
        v-if="me"
    >
        <BmmSingleMetadata
            v-model="form"
            @set="metadataIsSet = true"
            :languages="me.languages"
            :albums="me.albums"
        />
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
