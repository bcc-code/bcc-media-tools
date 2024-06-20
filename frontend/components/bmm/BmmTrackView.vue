<script lang="tsx" setup>
import {BMMTrack} from "~/src/gen/api/v1/api_pb";

defineProps<{
  track: BMMTrack;
}>();

const dateString = (date: Date) => {
  return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;
};
</script>

<template>
  <p class="flex cursor-pointer gap-2 rounded bg-slate-200 pl-2">
    <span v-if="track && track.publishedAt"> {{track.publishedAt ? dateString(track.publishedAt?.toDate()) : ''}}</span>
    <span class="rounded-r bg-slate-50 px-2">{{track.title}}</span>
  </p>
</template>
