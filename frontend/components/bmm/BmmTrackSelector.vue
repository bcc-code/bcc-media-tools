<script lang="tsx" setup>
import { BccFormLabel } from "@bcc-code/design-library-vue";
import { BMMTrack } from "../../src/gen/api/v1/api_pb";


const tracks = ref<BMMTrack[]>();

const props = defineProps<{
    album: string;
    label: string;
}>();

const api = useAPI();

onMounted(async () => {
  if (/^\d+$/.test(props.album)) {
    // Actual album
    tracks.value = (await api.getAlbumTracks({ albumId: props.album })).tracks;
  } else {
    // Podcast tag
    tracks.value = (await api.getPodcastTracks({ podcastTag: props.album, limit: 30 })).tracks;
  }
});

const value = defineModel<string>();

const clicked = ref(false);

const display = computed(() => {
    return tracks.value?.find((i) => i.id === value.value);
});

const dateString = (date: Date) => {
    return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;
};

const TrackView = (props: { track: BMMTrack }) => {
    const track = props.track;
    return (
        <p class="flex cursor-pointer gap-2 rounded bg-slate-50 pl-2">
            <span class="rounded-r bg-slate-200 px-2">{track.title}</span>
        </p>
    );
};
</script>

<template>
    <div>
        <BccFormLabel>
            {{ label }}
        </BccFormLabel>
        <div v-if="!clicked" class="flex">
            <TrackView
                @click="clicked = true"
                v-if="display"
                :track="display"
            />
            <p
                v-else
                class="flex cursor-pointer gap-2 rounded bg-slate-50"
                @click="clicked = true"
            >
                <span class="rounded bg-slate-200 px-2">Not selected</span>
            </p>
        </div>
        <div v-else class="flex h-48 flex-col gap-2 overflow-y-auto">
            <div v-for="t in tracks" class="flex">
                <TrackView
                    :track="t"
                    @click="
                        value = t.id;
                        clicked = false;
                    "
                />
            </div>
        </div>
    </div>
</template>