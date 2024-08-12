<script lang="ts" setup>
import { BmmEnvironment } from "~/src/gen/api/v1/api_pb";
import { BccInput, BccSelect } from "@bcc-code/design-library-vue";
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

const reset = () => {
    metadataIsSet.value = false;
    uploaded.value = false;
    selectedFiles.value = [];
    form.value = {
        title: "",
        environment: "prod",
    }
}
</script>

<template>
    <div
        class="mx-auto flex min-h-screen max-w-screen-md flex-col gap-4 rounded-lg bg-stone-300 p-4 text-black"
    >
        <template v-if="me && me.bmm && (me.bmm.podcasts.length > 0 || me.bmm.admin)">
            <template v-if="!uploaded">
                <BmmSingleMetadata
                    v-model="form"
                    @set="metadataIsSet = true"
                    :permissions="me.bmm"
                    :environment="selectedEnvironment"
                    v-if="!metadataIsSet"
                />
                <div v-if="metadataIsSet"
                    class="flex flex-col gap-4 p-4 transition"
                >
                    <h1 class="text-xl font-bold">Upload files for "{{metadata.title[0]}}"</h1>
                    <div v-for="file in selectedFiles" :key="file.file.name">
                        <BccSelect :class="[{
                            'hidden': !me.bmm.admin,
                        }]" :disabled="!me.bmm.admin" v-model="file.language" >
                            <option v-for="l in availableLanguages" :value="l">
                                {{ l }}
                            </option>
                        </BccSelect>
                        {{ file.file.name }} <button @click="selectedFiles.splice(selectedFiles.indexOf(file), 1)">
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="red" class="size-4">
                            <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
                        </svg>
                    </button>
                    </div>

                    <SelectFile
                        v-if="selectedFiles.length < 1 || me.bmm.admin"
                        v-model="selectedFiles"
                        :default-language="metadata.language[0]"
                        :accept-multiple="me.bmm.admin"
                    />

                    <FileUploader
                        v-model="selectedFiles"
                        :endpoint="config.public.grpcUrl + '/upload'"
                        :metadata="metadata"
                        @uploaded="uploaded = true"
                    />
                    <button class="rounded bg-slate-400 p-2" @click="metadataIsSet = false">Back</button>
                </div>
            </template>

            <template v-else>
                <div class="rounded-lg bg-green-500 p-4">
                    {{ $t("uploaded") }}
                </div>
                <button class="rounded bg-slate-400 p-2" @click="reset">Upload more</button>
            </template>
        </template>
        <template v-else> You don't have enough permissions </template>
    </div>
</template>
