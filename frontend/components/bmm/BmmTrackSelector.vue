<script lang="ts" setup>
import {
    BccFormLabel,
    BccSpinner,
    BccButton,
} from "@bcc-code/design-library-vue";
import type {
    BmmEnvironment,
    BMMTrack,
    LanguageList,
} from "~/src/gen/api/v1/api_pb";

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

const showOlderTracks = ref(false);
const olderTracks = computed(() => {
    if (!tracks.value?.length) return [];
    return tracks.value.filter((t) =>
        isBefore(t.publishedAt!.toDate(), new Date()),
    );
});
const futureTracks = computed(() => {
    if (!tracks.value?.length) return [];
    return tracks.value.filter((t) =>
        isAfter(t.publishedAt!.toDate(), new Date()),
    );
});

const filteredTracks = computed(() => {
    if (!tracks.value?.length) return [];

    if (!selectedTrack.value) {
        const tracksToSort = olderTracks.value
            ? tracks.value.slice(
                  0,
                  showOlderTracks.value
                      ? tracks.value.length
                      : futureTracks.value.length + 1,
              )
            : tracks.value;
        return tracksToSort.toSorted((a, b) => {
            if (!a.publishedAt || !b.publishedAt) return 0;
            return isBefore(a.publishedAt.toDate(), b.publishedAt.toDate())
                ? -1
                : 1;
        });
    }

    return tracks.value.filter((t) => t.id == selectedTrack.value!.id);
});

const languages = ref<LanguageList | undefined>();
watch(
    () => props.env,
    async (env) => {
        languages.value = await api.getLanguages({
            environment: env,
        });
    },
    { immediate: true },
);
</script>

<template>
    <div class="h-full overflow-y-auto">
        <BccFormLabel>
            {{ label }}
        </BccFormLabel>
        <div
            v-if="tracks && tracks.length > 0"
            class="relative mt-2 gap-2 space-y-2"
        >
            <BccButton
                v-if="olderTracks.length"
                @click="showOlderTracks = !showOlderTracks"
                type="button"
                variant="tertiary"
                class="w-full"
            >
                {{
                    showOlderTracks ? "Hide older tracks" : "Show older tracks"
                }}
            </BccButton>
            <TransitionGroup
                move-class="transition duration-300 ease-out"
                enter-active-class="transition duration-300 ease-out"
                enter-from-class="opacity-0 scale-95"
                enter-to-class="opacity-100 scale-100"
                leave-active-class="transition duration-300 ease-out absolute"
                leave-from-class="opacity-100 scale-100"
                leave-to-class="opacity-0 scale-95"
            >
                <BmmTrackView
                    v-for="t in filteredTracks"
                    :key="t.id"
                    :track="t"
                    :languages="languages"
                    @click="onTrackClick(t)"
                />
            </TransitionGroup>
        </div>

        <BccSpinner v-else size="sm" class="mx-auto" />
    </div>
</template>
