<script lang="ts" setup>
import type { BMMTrack, Language, LanguageList } from "~/src/gen/api/v1/api_pb";
import dayjs from "dayjs";

const props = defineProps<{
    track: BMMTrack;
    languages?: LanguageList;
}>();

const emit = defineEmits<{
    clickTranscription: [];
}>();

const formatter = new Intl.DateTimeFormat("en-GB", { timeZone: "Europe/Oslo" });
const dateString = (date: Date) => {
    return formatter.format(date);
};

function trackHasLanguage(language: Language) {
    return props.track.languages?.Languages.some(
        (l) => l.code === language.code,
    );
}

const { me } = useMe();
const availableLanguages = computed(() =>
    props.languages?.Languages.filter(
        (l) => me.value?.bmm?.languages.includes(l.code) ?? false,
    ),
);

const today = new Date();
const publishedAtToday = computed(() => {
    return dayjs().isSame(props.track.publishedAt?.toDate(), "day");
});

const isInPast = computed(() => {
    if (!props.track.publishedAt) return false;
    return (
        dayjs(props.track.publishedAt.toDate()).isBefore(today, "day") ||
        publishedAtToday.value
    );
});

const isInFuture = computed(() => {
    if (!props.track.publishedAt) return false;
    return dayjs(props.track.publishedAt.toDate()).isAfter(today, "day");
});
</script>

<template>
    <div
        class="grid w-full cursor-pointer grid-cols-[auto_1fr] grid-rows-[auto_1fr] overflow-clip rounded-md border border-neutral-300 bg-white shadow-xs hover:bg-neutral-50"
    >
        <span
            v-if="track && track.publishedAt"
            :class="[
                'row-span-2 border-r border-neutral-300 px-2 py-1 text-left tabular-nums',
                {
                    'text-neutral-400': isInPast,
                    'text-neutral-600': isInFuture,
                },
            ]"
        >
            {{ dateString(track.publishedAt.toDate()) }}
            <small v-if="publishedAtToday" class="block">Today</small>
        </span>
        <div class="col-start-2 flex grow justify-between gap-2 px-2 py-1">
            <p>{{ track.title }}</p>
            <UButton
                v-if="track.hasTranscriptions"
                size="xs"
                variant="link"
                type="button"
                @click.stop="emit('clickTranscription')"
            >
                Show transcript
            </UButton>
        </div>
        <div
            v-if="availableLanguages?.length"
            class="col-start-2 row-start-2 flex h-full flex-wrap gap-1 border-t border-neutral-300 px-2 py-2"
        >
            <img
                v-for="l in availableLanguages"
                :key="l.code"
                :title="languageCodeToName(l.code)"
                :src="'/images/flags/' + l.iconFile"
                class="inline h-4 rounded-sm border border-white shadow data-[disabled=true]:scale-90 data-[disabled=true]:opacity-25 data-[disabled=true]:grayscale"
                :alt="l.code"
                :data-disabled="!trackHasLanguage(l)"
            />
        </div>
    </div>
</template>
