<script lang="ts" setup>
import { BMMTrack } from "~/src/gen/api/v1/api_pb";

defineProps<{
    track: BMMTrack;
}>();

const dateString = (date: Date) => {
    return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;
};
</script>

<template>
    <div class="flex w-full cursor-pointer gap-2 rounded bg-slate-200 pl-2">
        <span v-if="track && track.publishedAt">
            {{ dateString(track.publishedAt.toDate()) }}</span
        >
        <span class="bg-slate-50 px-2 flex-grow">{{ track.title }}</span>
        <span class="rounded-r text-lg pr-2"><img v-for="l in track.languages?.Languages" :title="l.code" :src="'/images/flags/'+l.code+'.svg'" class="h-4 inline pl-1" :alt="l.code"/></span>
    </div>
</template>
