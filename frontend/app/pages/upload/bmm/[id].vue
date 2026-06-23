<script lang="ts" setup>
import { BmmEnvironment } from "~~/src/gen/api/v1/api_pb";
import type { BMMSingleForm, FileAndLanguage } from "~/utils/bmm";
import { usePermissionsLoading } from "~/utils/me";

useHead({
    title: "BMM Upload",
});

definePageMeta({
    layout: false,
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
        title: [routeParams.title ?? ""],
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
    <div class="mx-auto h-full w-full max-w-5xl p-4">
        <div class="mb-4 flex items-center justify-end gap-4">
            <ThemeSwitch />
            <LanguageSwitcher />
        </div>
        <div
            class="border-border-1 bg-surface-default flex h-full flex-col gap-4 rounded-2xl border p-4"
        >
            <DesignBanner
                v-if="!routeParamsAreValid"
                variant="error"
                icon="tabler:alert-triangle"
            >
                Invalid route parameters
            </DesignBanner>
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
                                <h1 class="text-2xl font-bold">
                                    Upload files for "{{ routeParams.title }}"
                                </h1>
                            </header>
                            <DesignCheckbox
                                v-model="forceOverride"
                                label="Replace transcription even if has been manually corrected"
                            />
                            <BmmSelectFile
                                v-if="selectedFiles.length < 1 || me.bmm.admin"
                                v-model="selectedFiles"
                                :default-language="metadata.language![0]!"
                                :accept-multiple="me.bmm.admin"
                                :environment="selectedEnvironment"
                            />
                            <BmmFileUploader
                                v-model="selectedFiles"
                                :endpoint="config.public.grpcUrl + '/upload'"
                                :metadata="metadata"
                                :forceOverride="forceOverride"
                                @uploaded="uploaded = true"
                            />
                        </div>
                    </template>
                    <template v-else>
                        <DesignBanner variant="success" icon="tabler:check">
                            {{ $t("uploaded") }}
                        </DesignBanner>
                        <p>You can now close this tab.</p>
                    </template>
                </template>
                <template v-else-if="permissionsLoading">Loading...</template>
                <template v-else> You don't have enough permissions </template>
            </template>
        </div>
    </div>
</template>
