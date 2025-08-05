<script lang="ts" setup>
import { BmmEnvironment } from "~/src/gen/api/v1/api_pb";
import type { BMMSingleForm, FileAndLanguage } from "~/utils/bmm";
import { usePermissionsLoading } from "~/utils/me";

useHead({
    title: "BMM Upload",
});

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({
        id: "upload_index",
        title: "upload",
    });
});

const form = ref<BMMSingleForm>({
    title: "",
    environment: "prod",
});

const metadataIsSet = ref(false);
const forceOverride = ref(false);

const selectedFiles = ref<FileAndLanguage[]>([]);

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

const formatter = new Intl.DateTimeFormat("en-GB", { timeZone: "Europe/Oslo" });
const dateString = (date: Date) => {
    return formatter.format(date);
};
</script>

<template>
    <div class="h-full p-4">
        <div
            class="mx-auto flex h-full max-w-screen-lg flex-col gap-4 rounded-2xl border border-neutral-300 bg-white p-4 text-black"
        >
            <template
                v-if="
                    me && me.bmm && (me.bmm.podcasts.length > 0 || me.bmm.admin)
                "
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
                            <h1 class="text-2xl font-bold">
                                Upload files for "{{ form.track.title }}" ({{
                                    dateString(
                                        form.track.publishedAt!.toDate(),
                                    )
                                }})
                            </h1>
                            <p class="text-xl">
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
                        <UCheckbox
                            v-model="forceOverride"
                            label="Replace transcription even if has been manually corrected"
                        />
                        <UTable
                            :key="selectedFiles.length"
                            :items="selectedFiles"
                            :columns="[
                                {
                                    accessorKey: 'language',
                                    header: 'Language',
                                },
                                {
                                    accessorKey: 'file.name',
                                    header: 'Name',
                                },
                                {
                                    id: 'actions',
                                    size: 50,
                                },
                            ]"
                            :data="selectedFiles"
                        >
                            <template #name-cell="item">
                                <div class="max-w-[420px] truncate">
                                    {{ item.getValue() }}
                                </div>
                            </template>
                            <template #language-cell="{ row }">
                                <LanguageSelector
                                    v-model="row.original.language"
                                    :class="{
                                        hidden: !me.bmm.admin,
                                    }"
                                    :disabled="!me.bmm.admin"
                                    :languages="me.bmm.languages"
                                    :env="selectedEnvironment"
                                    label=""
                                />
                            </template>
                            <template #actions-cell="{ row }">
                                <UButton
                                    @click="
                                        selectedFiles.splice(
                                            selectedFiles.indexOf(
                                                row.original as any,
                                            ),
                                            1,
                                        )
                                    "
                                    color="error"
                                    variant="link"
                                    square
                                >
                                    <Icon name="heroicons:trash" />
                                </UButton>
                            </template>
                        </UTable>
                        <SelectFile
                            v-if="selectedFiles.length < 1 || me.bmm.admin"
                            v-model="selectedFiles"
                            :default-language="metadata.language![0]!"
                            :accept-multiple="me.bmm.admin"
                        />
                        <FileUploader
                            v-model="selectedFiles"
                            :endpoint="config.public.grpcUrl + '/upload'"
                            :metadata="metadata"
                            :forceOverride="forceOverride"
                            @uploaded="uploaded = true"
                        />
                        <UButton
                            variant="ghost"
                            block
                            @click="metadataIsSet = false"
                        >
                            Back
                        </UButton>
                    </div>
                </template>
                <template v-else>
                    <UAlert
                        color="success"
                        variant="subtle"
                        :title="$t('uploaded')"
                    />
                    <UButton variant="soft" @click="reset" block>
                        Upload more
                    </UButton>
                </template>
            </template>
            <template v-else-if="permissionsLoading">Loading...</template>
            <template v-else> You don't have enough permissions </template>
        </div>
    </div>
</template>
