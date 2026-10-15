<script setup lang="ts">
import type { EditorialSession, PlayoutEvent } from "~~/src/gen/api/v1/api_pb";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate, timestampFromDate } from "@bufbuild/protobuf/wkt";

const { t } = useI18n();
const api = useAPI();
const perms = usePermissions();
const toaster = useToast();

useHead({ title: t("tools.editorial.title") });

const analytics = useAnalytics();
onMounted(() => analytics.page({ id: "editorial", title: "editorial" }));

const sessions = ref<EditorialSession[]>([]);
const loading = ref(true);

async function load() {
    loading.value = true;
    try {
        const res = await api.listEditorialSessions({});
        sessions.value = res.sessions;
    } finally {
        loading.value = false;
    }
}
onMounted(load);

const dateTimeFormatter = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Europe/Oslo",
    dateStyle: "medium",
    timeStyle: "short",
});
function formatDateTime(ts?: Timestamp): string {
    if (!ts) return "";
    return dateTimeFormatter.format(timestampDate(ts));
}

const createOpen = ref(false);
const newVxid = ref("");
const newTitle = ref("");
const creating = ref(false);

// A live meeting is logged in real time, before its recording is in
// Mediabanken; it needs the Playout event to sync names and songs from.
const newKind = ref<"recording" | "live">("recording");
const kindItems = computed(() => [
    { label: t("editorial.kindRecording"), value: "recording" },
    { label: t("editorial.kindLive"), value: "live" },
]);
const kindModel = computed<string>({
    get: () => newKind.value,
    set: (v) => (newKind.value = v === "live" ? "live" : "recording"),
});
const newEvent = ref<PlayoutEvent>();
// When the meeting started: Playout content from then on is synced, so a
// session opened late still picks up the beginning.
const newStartDate = ref("");
const newStartTime = ref("");
watch(createOpen, (open) => {
    if (!open) return;
    const now = new Date();
    newStartDate.value = osloDateInput(now);
    newStartTime.value = osloTimeInput(now);
});
const newStart = computed(() =>
    osloToDate(newStartDate.value, newStartTime.value),
);
const playoutEvents = ref<PlayoutEvent[]>([]);
const loadingPlayoutEvents = ref(false);
let playoutEventsLoaded = false;

watch([newKind, createOpen], async ([kind, open]) => {
    if (kind !== "live" || !open || playoutEventsLoaded) return;
    loadingPlayoutEvents.value = true;
    try {
        playoutEvents.value = (await api.listPlayoutEvents({})).events;
        playoutEventsLoaded = true;
    } catch {
        toaster.create({ title: t("editorial.importFailed"), type: "error" });
    } finally {
        loadingPlayoutEvents.value = false;
    }
});

const canCreate = computed(() =>
    newKind.value === "live"
        ? !!newEvent.value && !!newStart.value
        : !!newVxid.value.trim(),
);

async function create() {
    if (!canCreate.value) return;
    const live = newKind.value === "live";
    creating.value = true;
    try {
        const sess = await api.createEditorialSession({
            VXID: live ? "" : newVxid.value.trim(),
            title: newTitle.value.trim(),
            live,
            playoutEventId: live ? newEvent.value!.id : "",
            clockStart: live ? timestampFromDate(newStart.value!) : undefined,
        });
        createOpen.value = false;
        newVxid.value = "";
        newTitle.value = "";
        newEvent.value = undefined;
        await navigateTo(`/editorial/${sess.id}`);
    } catch {
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    } finally {
        creating.value = false;
    }
}
</script>

<template>
    <div v-if="!perms.canUseEditorial.value" class="py-16 text-center">
        <p class="text-body-2 text-text-muted">{{ t("noPermissions") }}</p>
    </div>

    <div v-else class="mx-auto w-full max-w-3xl px-4 py-8">
        <div class="mb-6 flex items-center justify-between">
            <h1 class="text-heading-2 text-text-default">
                {{ t("tools.editorial.title") }}
            </h1>
            <DesignButton
                v-if="perms.canRunLiveEditorial.value"
                icon="tabler:plus"
                @click="createOpen = true"
            >
                {{ t("editorial.newSession") }}
            </DesignButton>
        </div>

        <div v-if="loading" class="flex flex-col gap-2">
            <DesignSkeleton v-for="i in 4" :key="i" class="h-16 rounded-2xl" />
        </div>

        <p
            v-else-if="sessions.length === 0"
            class="text-body-3 text-text-hint py-16 text-center"
        >
            {{ t("editorial.noSessions") }}
        </p>

        <ul v-else class="flex flex-col gap-2">
            <li v-for="s in sessions" :key="s.id">
                <NuxtLink
                    :to="`/editorial/${s.id}`"
                    class="bg-surface-raise gradient-border shadow-resting ds-focus-ring hover:bg-surface-indent flex items-center justify-between gap-4 rounded-2xl px-4 py-3 transition-colors"
                >
                    <div class="min-w-0">
                        <div class="flex min-w-0 items-center gap-2">
                            <p class="text-title-2 text-text-default truncate">
                                {{
                                    s.title || s.VXID || t("editorial.untitled")
                                }}
                            </p>
                            <DesignBadge
                                v-if="s.kind === 'live'"
                                variant="warning"
                                class="shrink-0"
                            >
                                {{ t("editorial.live") }}
                            </DesignBadge>
                        </div>
                        <p class="text-caption-1 text-text-hint truncate">
                            {{ s.VXID || t("editorial.notInMediabanken") }} ·
                            {{ s.createdBy }}
                            <template v-if="s.updatedAt">
                                ·
                                {{
                                    t("editorial.lastUpdated", {
                                        date: formatDateTime(s.updatedAt),
                                    })
                                }}
                            </template>
                        </p>
                    </div>
                </NuxtLink>
            </li>
        </ul>
    </div>

    <DesignDialog
        v-model:open="createOpen"
        :title="t('editorial.createTitle')"
        :size="newKind === 'live' ? 'lg' : undefined"
    >
        <template #default="{ initialFocus }">
            <form class="flex flex-col gap-4" @submit.prevent="create">
                <DesignSegmentGroup
                    v-model="kindModel"
                    :items="kindItems"
                    class="border-border-1 border"
                />
                <DesignInput
                    v-if="newKind === 'recording'"
                    :ref="initialFocus"
                    v-model="newVxid"
                    :label="t('editorial.vxidLabel')"
                    placeholder="VX-123456"
                />
                <DesignInput
                    v-model="newTitle"
                    :label="t('editorial.titleLabel')"
                />
                <template v-if="newKind === 'live'">
                    <div class="flex flex-wrap gap-3">
                        <DesignInput
                            v-model="newStartDate"
                            type="date"
                            :label="t('editorial.meetingDate')"
                            class="min-w-40 grow"
                        />
                        <DesignInput
                            v-model="newStartTime"
                            placeholder="HH:MM"
                            :invalid="
                                !!newStartTime && !isClockTime(newStartTime)
                            "
                            :label="t('editorial.meetingStart')"
                            class="min-w-28 grow"
                        />
                    </div>
                    <p class="text-caption-1 text-text-muted">
                        {{ t("editorial.playoutEventLabel") }}
                    </p>
                    <EditorialPlayoutEventList
                        :events="playoutEvents"
                        :loading="loadingPlayoutEvents"
                        :selected-id="newEvent?.id"
                        @select="newEvent = $event"
                    />
                    <p class="text-caption-1 text-text-hint">
                        {{ t("editorial.liveHint") }}
                    </p>
                </template>
                <div class="flex justify-end gap-2">
                    <DesignButton
                        variant="secondary"
                        type="button"
                        @click="createOpen = false"
                    >
                        {{ t("editorial.cancel") }}
                    </DesignButton>
                    <DesignButton
                        type="submit"
                        :loading="creating"
                        :disabled="!canCreate"
                    >
                        {{
                            newKind === "live"
                                ? t("editorial.startLogging")
                                : t("editorial.create")
                        }}
                    </DesignButton>
                </div>
            </form>
        </template>
    </DesignDialog>
</template>
