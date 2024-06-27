<script lang="tsx" setup>
import { BccFormLabel } from "@bcc-code/design-library-vue";
import { BmmEnvironment, BMMTrack } from "~/src/gen/api/v1/api_pb";

const tracks = ref<BMMTrack[]>();

const props = defineProps<{
    album: string;
    label: string;
    env: BmmEnvironment;
}>();

const api = useAPI();

watch(
    [() => props.env, () => props.album],
    async ([env, album]) => {
        tracks.value = [];
        if (/^\d+$/.test(album)) {
            // Actual album
            tracks.value = (
                await api.getAlbumTracks({ albumId: album, environment: env })
            ).tracks;
        } else {
            // Podcast tag
            tracks.value = (
                await api.getPodcastTracks({
                    podcastTag: album,
                    environment: env,
                    limit: 30,
                })
            ).tracks;
        }
    },
    { immediate: true },
);

const selectedTrackId = defineModel<string>();
const selectedTrack = computed(() => {
    return tracks.value?.find((i) => i.id === selectedTrackId.value);
});
</script>

<template>
    <div>
        <BccFormLabel>
            {{ label }}
        </BccFormLabel>
        <div v-if="selectedTrack" class="flex">
            <BmmTrackView
                @click="selectedTrackId = ''"
                v-if="selectedTrack"
                :track="selectedTrack"
            />
        </div>
        <div
            v-else-if="tracks && tracks.length > 0"
            class="flex h-48 flex-col gap-2 overflow-y-auto"
        >
            <div v-for="t in tracks" class="flex">
                <BmmTrackView :track="t" @click="selectedTrackId = t.id" />
            </div>
        </div>

        <div v-else>
            <p class="flex cursor-pointer gap-2 rounded bg-slate-50">
                <span class="rounded bg-slate-200 px-2">Loading</span>
            </p>
        </div>
    </div>
</template>
