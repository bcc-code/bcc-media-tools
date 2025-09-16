<script lang="ts" setup>
import type {
    BMMTrack,
    Language,
    LanguageList,
} from "~~/src/gen/api/v1/api_pb";
import dayjs from "dayjs";

const props = defineProps<{
    track: BMMTrack;
    languages?: LanguageList;
    selected?: boolean;
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

const trackPublishedAt = computed(() =>
    timestampToDate(props.track.publishedAt),
);
const today = new Date();
const publishedAtToday = computed(() => {
    return dayjs().isSame(props.track.publishedAt?.nanos, "day");
});

const isInPast = computed(() => {
    if (!trackPublishedAt.value) return false;
    return (
        dayjs(trackPublishedAt.value).isBefore(today, "day") ||
        publishedAtToday.value
    );
});

const isInFuture = computed(() => {
    if (!trackPublishedAt.value) return false;
    return dayjs(trackPublishedAt.value).isAfter(today, "day");
});
</script>

<template>
    <div
        class="border-accented hover:bg-muted bg-default grid w-full cursor-pointer grid-cols-[auto_1fr] grid-rows-[auto_1fr] overflow-clip rounded-md border shadow-xs"
    >
        <span
            v-if="track && trackPublishedAt"
            :class="[
                'border-accented row-span-2 border-r px-2 py-1 text-left tabular-nums',
                {
                    'text-neutral-400': isInPast,
                    'text-neutral-600': isInFuture,
                },
            ]"
        >
            {{ dateString(trackPublishedAt) }}
            <small v-if="publishedAtToday" class="block">
                {{ $t("bmmUpload.today") }}
            </small>
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
                {{ $t("bmmUpload.showTranscription") }}
            </UButton>
        </div>
        <div
            v-if="availableLanguages?.length"
            class="border-accented col-start-2 row-start-2 flex h-full flex-wrap gap-1 border-t px-2 py-2"
        >
            <img
                v-for="l in availableLanguages"
                :key="l.code"
                :title="languageCodeToName(l.code)"
                :src="'/images/flags/' + l.iconFile"
                class="inline h-4 rounded-sm shadow data-[disabled=true]:scale-90 data-[disabled=true]:opacity-25 data-[disabled=true]:grayscale"
                :alt="l.code"
                :data-disabled="!trackHasLanguage(l)"
            />
        </div>
    </div>
</template>
