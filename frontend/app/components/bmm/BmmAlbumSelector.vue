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
const contentType = defineModel<"podcast" | "album">("contentType", {
    default: "album",
});

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
            value: y.year.toString(),
            count: y.count,
        }));
});

// DesignSelect is string-valued; bridge the numeric year.
const selectedYearStr = computed({
    get: () => String(selectedYear.value),
    set: (v) => {
        selectedYear.value = Number(v);
    },
});

const albumItems = computed(() => {
    return Object.entries(albums.value).map(([id, title]) => ({
        label: title,
        value: id,
    }));
});
</script>

<template>
    <div v-if="permissions.admin" class="flex flex-col gap-1">
        <label class="text-body-3 text-text-muted block">
            {{ $t("bmmUpload.type") }}
        </label>
        <DesignSelect
            v-model="selectedType"
            :items="[
                { label: $t('bmmUpload.album', 2), value: 'albums' },
                { label: $t('bmmUpload.podcast', 2), value: 'podcasts' },
            ]"
        />
    </div>

    <div v-if="selectedType == 'podcasts'" class="flex flex-col gap-1">
        <label class="text-body-3 text-text-muted block">
            {{ $t("bmmUpload.podcast") }}
        </label>
        <DesignSelect
            v-model="albumId"
            :disabled="permissions.podcasts.length < 2"
            :items="permissions.podcasts"
        />
    </div>

    <template v-else-if="selectedType == 'albums' && years">
        <div class="flex flex-col gap-1">
            <label class="text-body-3 text-text-muted block">
                {{ $t("bmmUpload.year") }}
            </label>
            <DesignSelect v-model="selectedYearStr" :items="yearItems">
                <template #item="{ item }">
                    <span>{{ (item as { label: string }).label }}</span>
                    <span class="text-text-hint text-caption-1 ml-3">
                        {{
                            $t(
                                "bmmUpload.albumCount",
                                {
                                    count: (
                                        item as unknown as { count: number }
                                    ).count,
                                },
                                (item as unknown as { count: number }).count,
                            )
                        }}
                    </span>
                </template>
            </DesignSelect>
        </div>
        <div v-if="Object.keys(albums).length" class="flex flex-col gap-1">
            <label class="text-body-3 text-text-muted block">
                {{ $t("bmmUpload.album") }}
            </label>
            <DesignSelect v-model="albumId" :items="albumItems" />
        </div>
    </template>
</template>
