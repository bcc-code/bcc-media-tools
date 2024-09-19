<script lang="ts" setup>
import { BccAlert, BccButton, BccTable } from "@bcc-code/design-library-vue";
import { BmmEnvironment } from "~/src/gen/api/v1/api_pb";
import type { BMMSingleForm, FileAndLanguage } from "~/utils/bmm";
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

const metadata = computed<Record<string, string[]>>(() => {
    let f: Record<string, string[]> = {
        title: [form.value.title],
        environment: [form.value.environment ?? "prod"],
    };
    if (form.value.track) f["trackId"] = [form.value.track.id];
    if (form.value.language) f["language"] = [form.value.language];

    return f;
});

const availableLanguages = ref<string[]>([]);
watch(
    [me],
    async ([newMe]) => {
        const newLanguages = newMe?.bmm?.languages;
        if (newLanguages && newLanguages.length > 0) {
            availableLanguages.value = newLanguages;
        } else {
            const languageList = await api.getLanguages({
                environment: selectedEnvironment.value,
            });
            availableLanguages.value = languageList.Languages.map(
                (l) => l.code,
            );
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
    };
};
</script>

<template>
    <div
        class="mx-auto my-8 flex h-full max-w-screen-md flex-col gap-4 rounded-2xl border border-on-secondary bg-white p-4 text-black"
    >
        <template
            v-if="me && me.bmm && (me.bmm.podcasts.length > 0 || me.bmm.admin)"
        >
            <template v-if="!uploaded">
                <!-- @vue-expect-error The component's `v-model` expects a form with the type `BMMSingleForm` -->
                <BmmSingleMetadata
                    v-if="!metadataIsSet"
                    v-model="form"
                    :permissions="me.bmm"
                    :environment="selectedEnvironment"
                    @set="metadataIsSet = true"
                />

                <div
                    v-if="metadataIsSet && form.track"
                    class="flex flex-col gap-4 p-4 transition"
                >
                    <header>
                        <h1 class="text-heading-xl">
                            Upload files for "{{ form.track.title }}"
                        </h1>
                        <p class="text-heading-md">
                            Existing languages:
                            <img
                                v-for="l in form.track.languages?.Languages"
                                :title="l.code"
                                :src="'/images/flags/' + l.iconFile"
                                class="ml-2 inline h-4 rounded-sm shadow-sm"
                                :alt="l.code"
                            />
                        </p>
                    </header>

                    <BccTable
                        :items="selectedFiles"
                        :columns="[
                            {
                                key: 'language',
                                text: 'Language',
                                sortable: false,
                            },
                            { key: 'file.name', text: 'Name' },
                            {
                                key: 'actions',
                                text: 'Actions',
                                sortable: false,
                            },
                        ]"
                    >
                        <template #item.file.name="{ item }">
                            <div
                                class="max-w-[420px] truncate"
                                :title="item.file.name"
                            >
                                {{ item.file.name }}
                            </div>
                        </template>
                        <template #item.language="{ item }">
                            <LanguageSelector
                                v-model="item.language"
                                :class="{
                                    hidden: !me.bmm.admin,
                                }"
                                :disabled="!me.bmm.admin"
                                :languages="me.bmm.languages"
                                :env="selectedEnvironment"
                                label=""
                            />
                        </template>
                        <template #item.actions="{ item }">
                            <BccButton
                                @click="
                                    selectedFiles.splice(
                                        selectedFiles.indexOf(item as any),
                                        1,
                                    )
                                "
                                context="danger"
                                variant="tertiary"
                            >
                                <Icon name="heroicons:trash" />
                            </BccButton>
                        </template>
                    </BccTable>

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

                    <BccButton
                        variant="secondary"
                        @click="metadataIsSet = false"
                    >
                        Back
                    </BccButton>
                </div>
            </template>

            <template v-else>
                <BccAlert context="success">
                    {{ $t("uploaded") }}
                </BccAlert>
                <BccButton variant="secondary" @click="reset">
                    Upload more
                </BccButton>
            </template>
        </template>
        <template v-else-if="permissionsLoading">Loading...</template>
        <template v-else> You don't have enough permissions </template>
    </div>
</template>
