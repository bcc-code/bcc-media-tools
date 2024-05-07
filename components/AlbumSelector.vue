<script lang="ts" setup>
import { BccSelect } from "@bcc-code/design-library-vue";
import {BMMYear} from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    usersAlbums: readonly string[];
}>();

const api = useAPI();
const value = defineModel<string>();

const years = ref<{[key: string]: BMMYear}>();
const selectedYear = ref<string>('podcasts');
const albums = ref<string[]>([]);

watch(selectedYear, async (newYear) => {
  if (newYear == "podcasts") {
    for (let a in props.usersAlbums) {
      albums.value.push(props.usersAlbums[a])
    }
  }
  let albumsRes = (await api.getAlbums({year: +newYear})).albums
  albums.value = []
  for (let a in albumsRes) {
    albums.value.push(albumsRes[a].title)
  }
});

onMounted(async () => {
  for (let a in props.usersAlbums) {
    albums.value.push(props.usersAlbums[a])
  }
  years.value = (await api.getYears({})).data
});

</script>

<template>
    <BccSelect v-model="selectedYear" :label="$t('Year')">
      <option value="podcasts">{{ $t("podcasts") }}</option>
      <option v-for="y in years" :value="y.year">
        {{ y.year }} ({{ y.count }})
      </option>
    </BccSelect>
    <BccSelect v-model="value" :label="$t('album')">
      <option disabled value="">{{ $t("selectAnOption") }}</option>
        <option v-for="a in albums" :value="a">
            {{ a }}
        </option>
    </BccSelect>
</template>
