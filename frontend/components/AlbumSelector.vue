<script lang="ts" setup>
import { BccSelect } from "@bcc-code/design-library-vue";
import {BmmEnvironment, BMMYear} from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    usersPodcasts: string[];
    env: BmmEnvironment
}>();

const api = useAPI();

const years = ref<{[key: string]: BMMYear}>();
const albums = ref<{[key: string]: string}>({});
const podcastTags = ref<string[]>(props.usersPodcasts);

const currentYear = (new Date()).getFullYear();
const selectedType = ref<string>('podcasts');
const selectedYear = ref<string>(currentYear.toString());
const value = defineModel<string>();

watch(() => props.env, async(env)=> {
  years.value = (await api.getYears({environment: env})).data
}, {immediate: true});

watch([selectedYear, () => props.env], async ([newYear, env]) => {
  value.value = "";
  albums.value = {}
  let albumsRes = (await api.getAlbums({year: parseInt(newYear), environment: env})).albums
  for (let a in albumsRes) {
    albums.value[albumsRes[a].id] = albumsRes[a].title;
  }
}, {immediate: true});

watch([() => props.env, selectedType], ([newEnv, newType])=> {
  if (newType === "podcasts")
    value.value = props.usersPodcasts[0];
  else
    value.value = "";
}, {immediate: true});


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
