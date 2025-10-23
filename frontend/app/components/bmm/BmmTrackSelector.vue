<script lang="ts" setup>
import type { BmmEnvironment, BMMTrack } from "~~/src/gen/api/v1/api_pb";
import dayjs from "dayjs";

const props = defineProps<{
    album: string;
    label: string;
    env: BmmEnvironment;
}>();

defineEmits<{
    transcription: [id: string];
}>();

const api = useAPI();

const isYearAlbum = (album: string) => /^\d+$/.test(album);

const {
    data: tracks,
    status,
    error,
} = useAsyncData(
    () => `${props.env}:${props.album}:tracks`,
    () => {
        if (isYearAlbum(props.album)) {
            return api.getAlbumTracks({
                albumId: props.album,
                environment: props.env,
            });
        } else {
            return api.getPodcastTracks({
                podcastTag: props.album,
                environment: props.env,
                limit: 30,
            });
        }
    },
    { transform: (data) => data.tracks },
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
    return tracks.value.filter((t) => {
        return dayjs(timestampToDate(t.publishedAt)).isBefore(dayjs(), "day");
    });
});
const futureTracks = computed(() => {
    if (!tracks.value?.length) return [];
    return tracks.value.filter(
        (t) =>
            dayjs(timestampToDate(t.publishedAt)).isAfter(dayjs(), "day") ||
            dayjs(timestampToDate(t.publishedAt)).isSame(dayjs(), "day"),
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
                      : futureTracks.value.length,
              )
            : tracks.value;
        return tracksToSort.toSorted((a, b) => {
            if (!a.publishedAt || !b.publishedAt) return 0;
            return dayjs(timestampToDate(a.publishedAt)).isBefore(
                timestampToDate(b.publishedAt),
                "day",
            )
                ? -1
                : 1;
        });
    }

    return tracks.value.filter((t) => t.id == selectedTrack.value!.id);
});

const { data: languages } = useAsyncData(
    () => `${props.env}:languages`,
    () => api.getLanguages({ environment: props.env }),
);

const transcriptionTrack = ref<BMMTrack>();
</script>

<template>
    <div class="h-full overflow-y-auto">
        <p>
            {{ label }}
        </p>
        <div
            v-if="status == 'success' && tracks && tracks.length"
            class="relative mt-2 gap-2 space-y-2"
        >
            <TransitionGroup
                move-class="transition duration-300 ease-out"
                enter-active-class="transition duration-300 ease-out"
                enter-from-class="opacity-0 scale-95"
                enter-to-class="opacity-100 scale-100"
                leave-active-class="transition duration-300 ease-out absolute"
                leave-from-class="opacity-100 scale-100"
                leave-to-class="opacity-0 scale-95"
            >
                <UButton
                    v-if="olderTracks.length && !selectedTrack"
                    @click="showOlderTracks = !showOlderTracks"
                    type="button"
                    variant="link"
                    block
                >
                    {{
                        showOlderTracks
                            ? $t("bmmUpload.hideOlderTracks")
                            : $t("bmmUpload.showOlderTracks")
                    }}
                </UButton>
                <BmmTrackView
                    v-for="t in filteredTracks"
                    :key="t.id"
                    :track="t"
                    :languages="languages"
                    :selected="selectedTrack?.id == t.id"
                    @click="onTrackClick(t)"
                    @click-transcription="transcriptionTrack = t"
                />
            </TransitionGroup>
        </div>

        <div v-else-if="status == 'pending'" class="flex justify-center">
            <Icon name="svg-spinners:bars-rotate-fade" class="size-8" />
        </div>

        <p
            v-else-if="tracks && !tracks.length"
            class="text-dimmed my-8 text-center text-sm"
        >
            {{ $t("bmmUpload.noTracks") }}
        </p>

        <pre>{{ error }}</pre>
    </div>
    <BmmTranscriptionDialog v-model:track="transcriptionTrack" :env="env" />
</template>
