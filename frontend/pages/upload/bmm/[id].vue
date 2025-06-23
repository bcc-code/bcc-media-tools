<script lang="ts" setup>
import {
    BccAlert,
    BccButton,
    BccTable,
    BccCheckbox,
} from "@bcc-code/design-library-vue";
import { BmmEnvironment } from "~/src/gen/api/v1/api_pb";
import type { BMMSingleForm, FileAndLanguage } from "~/utils/bmm";
import { usePermissionsLoading } from "~/utils/me";

useHead({
    title: "BMM Upload",
});

type RouteParams = {
    id?: string;
    lang?: string | string[];
    env?: string;
    title?: string;
};
const requiredRouteParams: (keyof RouteParams)[] = ["id", "lang"];

const route = useRoute("upload-bmm-id");
const routeParams: RouteParams = { id: route.params.id, ...route.query };
const lang =
    routeParams.lang instanceof Array ? routeParams.lang[0] : routeParams.lang;

const routeParamsAreValid = computed(() =>
    requiredRouteParams.map((k) => k in routeParams).every(Boolean),
);

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({
        id: "upload_redirect",
        title: "bmm upload",
        meta: {
            trackId: routeParams.id,
            language: lang,
            environment: routeParams.env,
        },
    });
});

const form = ref<BMMSingleForm>({
    title: "",
    environment: "prod",
    language: lang,
});

const selectedEnvironment = computed(() => {
    return routeParams.env === "int"
        ? BmmEnvironment.Integration
        : BmmEnvironment.Production;
});

const forceOverride = ref(false);

const selectedFiles = ref<FileAndLanguage[]>([]);

const { me } = useMe();
const config = useRuntimeConfig();

const permissionsLoading = usePermissionsLoading();

const metadata = computed<Record<string, string[]>>(() => {
    let f: Record<string, string[]> = {
        title: [form.value.title],
        environment: [routeParams.env ?? "prod"],
    };
    if (routeParams.id) f["trackId"] = [routeParams.id];
    if (routeParams.lang)
        f["language"] =
            routeParams.lang instanceof Array
                ? routeParams.lang
                : [routeParams.lang];

    return f;
});

const uploaded = ref(false);
</script>

<template>
    <div class="h-full p-4">
        <div
            class="mx-auto flex h-full max-w-screen-lg flex-col gap-4 rounded-2xl border border-on-secondary bg-white p-4 text-black"
        >
            <BccAlert v-if="!routeParamsAreValid" context="danger">
                <div class="flex items-center gap-2">
                    <Icon
                        name="heroicons:exclamation-triangle"
                        class="text-lg"
                    />
                    Invalid route parameters
                </div>
            </BccAlert>
            <template v-else>
                <template
                    v-if="
                        me &&
                        me.bmm &&
                        (me.bmm.podcasts.length > 0 || me.bmm.admin)
                    "
                >
                    <template v-if="!uploaded">
                        <div class="flex flex-col gap-4 p-4 transition">
                            <header v-if="routeParams.title">
                                <h1 class="text-heading-xl">
                                    Upload files for "{{ routeParams.title }}"
                                </h1>
                            </header>
                            <BccCheckbox
                                v-model="forceOverride"
                                label="Replace transcription even if has been manually corrected"
                            />
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
                                                selectedFiles.indexOf(
                                                    item as any,
                                                ),
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
                                :forceOverride="forceOverride"
                                @uploaded="(uploaded = true)"
                            />
                        </div>
                    </template>
                    <template v-else>
                        <BccAlert context="success">
                            <div class="flex items-center gap-2">
                                <Icon
                                    name="heroicons:check"
                                    class="text-lg opacity-50"
                                />
                                {{ $t("uploaded") }}
                            </div>
                        </BccAlert>
                        <p>You can now close this tab.</p>
                    </template>
                </template>
                <template v-else-if="permissionsLoading">Loading...</template>
                <template v-else> You don't have enough permissions </template>
            </template>
        </div>
    </div>
</template>
