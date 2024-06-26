<script lang="ts" setup>
import { BmmEnvironment } from "~/src/gen/api/v1/api_pb";
import { BccSelect } from "@bcc-code/design-library-vue";
import type { FileAndLanguage } from "~/utils/bmm";

const form = ref<BMMSingleForm>({
    title: "",
    environment: "prod",
});

const metadataIsSet = ref(false);

const selectedFiles = ref<FileAndLanguage[]>([]);

const api = useAPI();
const { me } = useMe();
const config = useRuntimeConfig();

const selectedEnvironment = computed(() => {
    return form.value.environment === "int"
        ? BmmEnvironment.Integration
        : BmmEnvironment.Production;
});

const metadata = computed(() => {
    return {
        title: [form.value.title],
        language: [form.value.language],
        trackId: [form.value.trackId?.toString() ?? ""],
        environment: [form.value.environment ?? "prod"],
        firstFile: [selectedFiles.value[0]?.file?.name ?? ""], // just for debugging
    } as { [key: string]: readonly string[] };
});

const availableLanguages = ref<string[]>([]);
watch(
    [me],
    async ([newMe]) => {
        const newLanguages = newMe?.bmm?.languages;
        if (newLanguages && newLanguages.length > 0) {
            availableLanguages.value = newLanguages;
        } else {
            availableLanguages.value = (
                await api.getLanguages({
                    environment: selectedEnvironment.value,
                })
            ).Languages;
        }
    },
    { immediate: true },
);

const uploaded = ref(false);
</script>

<template>
    <div
        class="mx-auto flex min-h-screen max-w-screen-md flex-col gap-4 rounded-lg bg-stone-300 p-4 text-black"
        v-if="me && me.bmm"
    >
        <template v-if="me.bmm && (me.bmm.podcasts.length > 0 || me.bmm.admin)">
            <template v-if="!uploaded">
                <BmmSingleMetadata
                    v-model="form"
                    @set="metadataIsSet = true"
                    :permissions="me.bmm"
                    :environment="selectedEnvironment"
                />
                <div
                    class="flex flex-col gap-4 p-4 transition"
                    :class="[
                        {
                            'pointer-events-none opacity-50': false,
                        },
                    ]"
                >
                    <h3 class="text-lg font-bold">Upload File</h3>

                    <div v-for="file in selectedFiles" :key="file.file.name">
                        <BccSelect v-model="file.language">
                            <option v-for="l in availableLanguages" :value="l">
                                {{ l }}
                            </option>
                        </BccSelect>
                        {{ file.file.name }}
                    </div>
                    <SelectFile v-model="selectedFiles" />
                    <FileUploader
                        v-for="(_, i) in selectedFiles"
                        v-model="selectedFiles[i]"
                        :endpoint="config.public.grpcUrl + '/upload'"
                        :metadata="metadata"
                        @uploaded="uploaded = true"
                    />
                    <div>
                        {{ selectedEnvironment }}
                        {{ metadata }}
                    </div>
                </div>
            </template>
            <template v-else>
                <div class="rounded-lg bg-green-500 p-4">
                    {{ $t("uploaded") }}
                </div>
            </template>
        </template>
        <template v-else> You don't have enough permissions </template>
    </div>
</template>
