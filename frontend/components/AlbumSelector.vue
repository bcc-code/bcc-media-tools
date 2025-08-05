<script lang="ts" setup>
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
const selectedType = ref<"podcasts" | "albums">("albums");
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
            const alb = albumsRes[a];
            if (alb) {
                albums.value[alb.id] = alb.title;
            }
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
    <UFormField v-if="permissions.admin" :label="$t('Type')">
        <USelect
            v-model="selectedType"
            :items="[
                {
                    label: $t('Albums'),
                    value: 'albums',
                },
                {
                    label: $t('Podcasts'),
                    value: 'podcasts',
                },
            ]"
            size="lg"
            class="w-full"
        />
    </UFormField>

    <UFormField v-if="selectedType == 'podcasts'" :label="$t('Podcast')">
        <USelect
            v-model="albumId"
            :disabled="permissions.podcasts.length < 2"
            :items="permissions.podcasts"
            size="lg"
            class="w-full"
        />
    </UFormField>

    <template v-else-if="selectedType == 'albums' && years">
        <UFormField v-model="selectedYear" :label="$t('Year')">
            <USelect
                value-key="value"
                label-key="label"
                :items="
                    Object.values(years)
                        .map((y) => ({
                            label: `${y.year} (${y.count})`,
                            value: y.year,
                        }))
                        .sort((a, b) => b.value - a.value)
                "
                size="lg"
                class="w-full"
            />
        </UFormField>
    </template>
</template>
