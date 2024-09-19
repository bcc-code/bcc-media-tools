<script lang="ts" setup>
import { BccSelect } from "@bcc-code/design-library-vue";
import {
    BmmEnvironment,
    BMMPermission,
    BMMYear,
} from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    permissions: BMMPermission;
    env: BmmEnvironment;
}>();

const api = useAPI();

const years = ref<{ [key: string]: BMMYear }>();
const albums = ref<{ [key: string]: string }>({});

const currentYear = new Date().getFullYear();
const selectedType = ref("podcasts");
const selectedYear = ref(currentYear.toString());
const albumId = defineModel<string>();

watch(
    () => props.env,
    async (env) => {
        years.value = (await api.getYears({ environment: env })).data;
    },
    { immediate: true },
);

watch(
    [selectedYear, () => props.env],
    async ([newYear, env]) => {
        albumId.value = "";
        albums.value = {};
        let albumsRes = (
            await api.getAlbums({ year: parseInt(newYear), environment: env })
        ).albums;
        for (let a in albumsRes) {
            albums.value[albumsRes[a].id] = albumsRes[a].title;
        }
    },
    { immediate: true },
);

watch(
    [selectedType, () => props.permissions.podcasts],
    async ([newType, newPodcasts]) => {
        await nextTick();
        albumId.value = newType === "podcasts" ? newPodcasts[0] : "";
    },
    { immediate: true },
);
</script>

<template>
    <BccSelect
        v-if="permissions.admin"
        v-model="selectedType"
        :label="$t('Type')"
    >
        <option value="podcasts">{{ $t("podcasts") }}</option>
        <option value="albums">{{ $t("albums") }}</option>
    </BccSelect>

    <BccSelect
        v-if="selectedType == 'podcasts'"
        :disabled="permissions.podcasts.length < 2"
        v-model="albumId"
        :label="$t('Podcast')"
    >
        <option v-for="p in permissions.podcasts" :value="p">
            {{ p }}
        </option>
    </BccSelect>

    <template v-else>
        <BccSelect v-model="selectedYear" :label="$t('Year')">
            <option v-for="y in years" :value="y.year">
                {{ y.year }} ({{ y.count }})
            </option>
        </BccSelect>
        <BccSelect v-model="albumId" :label="$t('album')">
            <option disabled value="">{{ $t("selectAnOption") }}</option>
            <option v-for="(title, key) in albums" :value="key">
                {{ title }}
            </option>
        </BccSelect>
    </template>
</template>
