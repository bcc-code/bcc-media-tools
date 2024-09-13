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
    <div
        class="flex w-full cursor-pointer items-center overflow-clip rounded-md border border-on-primary bg-primary shadow-sm hover:border-neutral-300"
    >
        <span
            v-if="track && track.publishedAt"
            class="w-24 bg-secondary px-2 py-1 text-left text-secondary"
        >
            {{ dateString(track.publishedAt.toDate()) }}
        </span>
        <span class="grow px-2 py-1 text-primary">
            {{ track.title }}
        </span>
        <span class="flex h-full gap-1 bg-secondary px-2 py-2">
            <img
                v-for="l in track.languages?.Languages"
                :title="l.code"
                :src="'/images/flags/' + l.code + '.svg'"
                class="inline h-4 rounded-sm border border-white shadow-sm"
                :alt="l.code"
            />
        </span>
    </div>
</template>
