<script lang="ts" setup>
import { BmmEnvironment } from "~~/src/gen/api/v1/api_pb";
import type { BMMPermission, BMMYear } from "~~/src/gen/api/v1/api_pb";

const props = defineProps<{
    permissions: BMMPermission;
    env: BmmEnvironment;
}>();

const api = useAPI();

const years = ref<{ [key: string]: BMMYear }>();
const albums = ref<{ [key: string]: string }>({});

const currentYear = new Date().getFullYear();
const selectedYear = ref(currentYear);
const albumId = defineModel<string>();
const contentType = defineModel<"podcast" | "album">("contentType", { default: "album" });

const selectedType = computed({
    get: () => (contentType.value === "podcast" ? "podcasts" : "albums"),
    set: (v) => (contentType.value = v === "podcasts" ? "podcast" : "album"),
});

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
            await api.getAlbums({ year: newYear, environment: env })
        ).albums;
        albumsRes.forEach((a) => {
            albums.value[a.id] = a.title;
        });
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

const yearItems = computed(() => {
    if (!years.value) return [];
    return Object.values(years.value)
        .sort((a, b) => b.year - a.year)
        .map((y) => ({
            label: y.year.toString(),
            value: y.year,
            count: y.count,
        }));
});

const albumItems = computed(() => {
    return Object.entries(albums.value).map(([id, title]) => ({
        label: title,
        value: id,
    }));
});
</script>

<template>
    <UFormField v-if="permissions.admin" :label="$t('bmmUpload.type')">
        <USelect
            v-model="selectedType"
            :items="[
                {
                    label: $t('bmmUpload.album', 2),
                    value: 'albums',
                },
                {
                    label: $t('bmmUpload.podcast', 2),
                    value: 'podcasts',
                },
            ]"
            size="lg"
            class="w-full"
        />
    </UFormField>

    <UFormField
        v-if="selectedType == 'podcasts'"
        :label="$t('bmmUpload.podcast')"
    >
        <USelect
            v-model="albumId"
            :disabled="permissions.podcasts.length < 2"
            :items="permissions.podcasts"
            size="lg"
            class="w-full"
        />
    </UFormField>

    <template v-else-if="selectedType == 'albums' && years">
        <UFormField :label="$t('bmmUpload.year')">
            <USelect
                v-model="selectedYear"
                :items="yearItems"
                size="lg"
                class="w-full"
            >
                <template #item-trailing="{ item }">
                    <span class="text-dimmed">
                        {{
                            $t(
                                "bmmUpload.albumCount",
                                { count: item.count },
                                item.count,
                            )
                        }}
                    </span>
                </template>
            </USelect>
        </UFormField>
        <UFormField
            v-if="Object.keys(albums).length"
            :label="$t('bmmUpload.album')"
        >
            <USelect
                v-model="albumId"
                :items="albumItems"
                size="lg"
                class="w-full"
            />
        </UFormField>
    </template>
</template>
