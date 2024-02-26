<script lang="tsx" setup>
import { BccFormLabel } from "@bcc-code/design-library-vue";

const tracks = ref<Track[]>();

const props = defineProps<{
    album: string;
    label: string;
}>();

onMounted(async () => {
    tracks.value = await $fetch("/api/bmm/tracks/" + props.album, {
        method: "GET",
    });
});

const value = defineModel<number>();

const clicked = ref(false);

const display = computed(() => {
    return tracks.value?.find((i) => i.id === value.value);
});

const dateString = (date: Date) => {
    return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;
};

const TrackView = (props: { track: Track }) => {
    const track = props.track;
    return (
        <p class="flex cursor-pointer gap-2 rounded bg-slate-50 pl-2">
            <span>{dateString(new Date(track.published_at))}</span>
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
        </div>
        <div v-else class="flex flex-col gap-2">
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
