<script lang="tsx" setup>
import { BccFormLabel } from "@bcc-code/design-library-vue";
import {BmmEnvironment, BMMTrack} from "~/src/gen/api/v1/api_pb";


const tracks = ref<BMMTrack[]>();

const props = defineProps<{
    album: string;
    label: string;
    env: BmmEnvironment;
}>();

const api = useAPI();

watch(() => props.env, async (env) => {
  if (/^\d+$/.test(props.album)) {
    // Actual album
    tracks.value = (await api.getAlbumTracks({ albumId: props.album, environment: env })).tracks;
  } else {
    // Podcast tag
    tracks.value = (await api.getPodcastTracks({ podcastTag: props.album, environment: env, limit: 30 })).tracks;
  }
}, {immediate: true})

const selectedTrackId = defineModel<string>();
const selectedTrack = computed(() => {
    return tracks.value?.find((i) => i.id === selectedTrackId.value);
});

const dateString = (date: Date) => {
    return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;
};

const TrackView = (props: { track: BMMTrack }) => {
    const track = props.track;
    return (
        <p class="flex cursor-pointer gap-2 rounded bg-slate-200 pl-2">
          <span v-if="track.publishedAt"> {track.publishedAt ? dateString(track.publishedAt?.toDate()) : ''}</span>
          <span class="rounded-r bg-slate-50 px-2">{track.title}</span>
        </p>
    );
};
</script>

<template>
  <div>
    <BccFormLabel>
      {{ label }}
    </BccFormLabel>
        <div v-if="selectedTrack" class="flex">
            <TrackView
                @click="selectedTrackId = ''"
                v-if="selectedTrack"
                :track="selectedTrack"
            />
        </div>
        <div v-else-if="tracks && tracks.length > 0" class="flex h-48 flex-col gap-2 overflow-y-auto">
            <div v-for="t in tracks" class="flex">
                <TrackView
                    :track="t"
                    @click="selectedTrackId = t.id;"
                />
            </div>
        </div>

        <div v-else>
            <p class="flex cursor-pointer gap-2 rounded bg-slate-50">
                <span class="rounded bg-slate-200 px-2">Not selected</span>
            </p>
        </div>

    </div>
</template>
