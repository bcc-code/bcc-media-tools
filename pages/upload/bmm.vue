<script lang="ts" setup>
const form = ref<BMMSingleForm>({
    title: "",
});

const metadataIsSet = ref(false);

const selectedFile = ref<File | null>(null);

const { me } = useMe();

const metadata = computed(() => {
    return {
        title: [form.value.title],
        language: [form.value.language],
        trackId: [form.value.trackId?.toString() ?? ""],
    } as { [key: string]: readonly string[] };
});

const uploaded = ref(false);
</script>

<template>
    <div
        class="mx-auto flex min-h-screen max-w-screen-md flex-col gap-4 rounded-lg bg-stone-300 p-4 text-black"
        v-if="me"
    >
        <template v-if="!uploaded">
            <BmmSingleMetadata
                v-model="form"
                @set="metadataIsSet = true"
                :languages="me.bmm.languages"
                :albums="me.bmm.albums"
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
                    :metadata="metadata"
                    @uploaded="uploaded = true"
                />
            </div>
        </template>
        <template v-else>
            <div class="rounded-lg bg-green-500 p-4">{{ $t("uploaded") }}</div>
        </template>
    </div>
</template>
