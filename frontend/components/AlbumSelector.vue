<script lang="ts" setup>
import { BccSelect } from "@bcc-code/design-library-vue";
import {BMMYear} from "../src/gen/api/v1/api_pb";

const props = defineProps<{
    usersAlbums: readonly string[];
}>();

const api = useAPI();

const years = ref<{[key: string]: BMMYear}>();
const albums = ref<{[key: string]: string}>({});
const podcastTags = ref<string[]>(['fra-kaare']);

const selectedYear = ref<string>('2024');
const selectedType = ref<string>('podcasts');
const value = defineModel<string>()

watch(selectedYear, async (newYear) => {
  albums.value = {}
  let albumsRes = (await api.getAlbums({year: +newYear})).albums
  for (let a in albumsRes) {
    albums.value[albumsRes[a].id] = albumsRes[a].title;
  }
}, {immediate: true});

onMounted(async () => {
  years.value = (await api.getYears({})).data
});

</script>

<template>
  <BccSelect v-model="selectedType" :label="$t('Type')">
    <option value="podcasts">{{ $t("podcasts") }}</option>
    <option value="albums">{{ $t("albums") }}</option>
  </BccSelect>

  <template v-if="selectedType == 'podcasts'">
    <BccSelect v-model="value" :label="$t('Podcast')">
      <option v-for="p in podcastTags" :value="p">
        {{ p }}
      </option>
    </BccSelect>
  </template>

  <template v-else>
    <BccSelect v-model="selectedYear" :label="$t('Year')">
      <option v-for="y in years" :value="y.year">
        {{ y.year }} ({{ y.count }})
      </option>
    </BccSelect>
    <BccSelect v-model="value" :label="$t('album')">
      <option disabled value="">{{ $t("selectAnOption") }}</option>
      <option v-for="(title, key) in albums" :value="key">
        {{ title }}
      </option>
    </BccSelect>
  </template>

</template>
