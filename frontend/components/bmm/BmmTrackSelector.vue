<script lang="ts" setup>
import {
    BccButton,
    BccFormLabel,
    BccTable,
    BccSpinner,
} from "@bcc-code/design-library-vue";
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

const selectedTrack = defineModel<BMMTrack>();

function onTrackClick(track: BMMTrack) {
    if (selectedTrack.value && selectedTrack.value.id == track.id) {
        selectedTrack.value = undefined;
    } else {
        selectedTrack.value = track;
    }
}

const filteredTracks = computed(() => {
    if (!tracks.value?.length) return [];

    if (!selectedTrack.value) {
        return tracks.value;
    }

    return tracks.value.filter((t) => t.id == selectedTrack.value!.id);
});
</script>

<template>
    <div class="h-96 overflow-y-auto">
        <BccFormLabel>
            {{ label }}
        </BccFormLabel>
        <div
            v-if="tracks && tracks.length > 0"
            class="relative mt-2 gap-2 space-y-2"
        >
            <TransitionGroup
                move-class="transition duration-600 ease-out"
                enter-active-class="transition duration-600 ease-out"
                enter-from-class="opacity-0 scale-95"
                enter-to-class="opacity-100 scale-100"
                leave-active-class="transition duration-600 ease-out absolute"
                leave-from-class="opacity-100 scale-100"
                leave-to-class="opacity-0 scale-95"
            >
                <BmmTrackView
                    v-for="t in filteredTracks"
                    :key="t.id"
                    :track="t"
                    @click="onTrackClick(t)"
                />
            </TransitionGroup>
        </div>

        <BccSpinner v-else size="sm" class="mx-auto" />
    </div>
</template>
