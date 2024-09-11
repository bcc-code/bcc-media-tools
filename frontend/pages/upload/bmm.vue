<script lang="ts" setup>
import { BmmEnvironment } from "~/src/gen/api/v1/api_pb";
import { BccInput, BccSelect } from "@bcc-code/design-library-vue";
import type { FileAndLanguage } from "~/utils/bmm";
import { usePermissionsLoading } from "~/utils/me";

const form = ref<BMMSingleForm>({
    title: "",
    environment: "prod",
});

const metadataIsSet = ref(false);

const selectedFiles = ref<FileAndLanguage[]>([]);

const api = useAPI();
const { me } = useMe();
const config = useRuntimeConfig();

const permissionsLoading = usePermissionsLoading();

const selectedEnvironment = computed(() => {
    return form.value.environment === "int"
        ? BmmEnvironment.Integration
        : BmmEnvironment.Production;
});

const metadata = computed(() => {
    return {
        title: [form.value.title],
        language: [form.value.language],
        trackId: [form.value.track.id],
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
        track: undefined,
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
                    <h1 class="text-xl font-bold">Upload files for "{{form.track.title}}"</h1>
                    <h2 class="text-lg font-bold">Existing languages: <span class="rounded-r text-lg pr-2"><span v-for="l in form.track.languages?.Languages" :title="l.code">{{l.flagEmoji}}</span></span>
                    </h2>
                    <div v-for="file in selectedFiles" :key="file.file.name">
                        <BccSelect :class="[{
                            'hidden': !me.bmm.admin,
                        }]" :disabled="!me.bmm.admin" v-model="file.language" >
                            <option v-for="l in availableLanguages" :value="l">
                                {{ l }}
                            </option>
                        </BccSelect>
                        {{ file.file.name }} <button @click="selectedFiles.splice(selectedFiles.indexOf(file), 1)">
                        <Icon :style="{color: 'red'}" name="heroicons:trash" />
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
        <template v-else-if="permissionsLoading">Loading...</template>
        <template v-else> You don't have enough permissions </template>
    </div>
</template>
