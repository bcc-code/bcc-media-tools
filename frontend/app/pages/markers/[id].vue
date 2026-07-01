<script setup lang="ts">
import { markerTypeMeta, sortMarkers } from "~/utils/markers";
import type { Marker } from "~/utils/markers";

const route = useRoute();
const vxId = computed(() => String(route.params.id ?? ""));

const { t } = useI18n();
useHead({ title: () => `Markers · ${vxId.value}` });

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({
        id: "markers_id",
        title: "markers",
        meta: { id: vxId.value },
    });
});

const api = useAPI();
const { markers, add, update, remove, restore, save, saving } =
    useMarkers(vxId);

// ---- Video preview (reuses the existing getPreview RPC, like shorts) --------
const { data: videoUrl, status: videoStatus } = useAsyncData(
    () => `markers-preview:${vxId.value}`,
    () => api.getPreview({ VXID: vxId.value }).then((r) => r.url),
);

const videoElement = useTemplateRef("videoElement");
const videoDuration = ref(0);
const currentTime = ref(0);

useEventListener(videoElement, "loadedmetadata", () => {
    videoDuration.value = videoElement.value?.duration ?? 0;
});
useEventListener(videoElement, "timeupdate", () => {
    currentTime.value = videoElement.value?.currentTime ?? 0;
});

// The timeline needs a span even before/without a loaded video: fall back to
// the last marker's out-point (plus headroom) so blocks still render.
const effectiveDuration = computed(() => {
    if (videoDuration.value > 0) return videoDuration.value;
    const lastEnd = markers.value.reduce((max, m) => Math.max(max, m.end), 0);
    return Math.max(lastEnd + 60, 300);
});

function seek(seconds: number) {
    if (!videoElement.value) return;
    videoElement.value.currentTime = seconds;
}

// ---- Selection + filtering --------------------------------------------------
const selectedId = ref<string>();
const selectedMarker = computed(() =>
    markers.value.find((m) => m.id === selectedId.value),
);

const visibleMarkers = computed(() => sortMarkers(markers.value));

// Markers whose range contains the playhead — shown as an on-video overlay.
const activeMarkers = computed(() =>
    sortMarkers(markers.value).filter(
        (m) => currentTime.value >= m.start && currentTime.value <= m.end,
    ),
);

// ---- Mutations --------------------------------------------------------------
function addMarker() {
    const start = currentTime.value;
    const end = Math.min(start + 5, effectiveDuration.value);
    const marker = add({ start, end, type: "name-super" });
    selectedId.value = marker.id;
}

function onUpdate(patch: Partial<Omit<Marker, "id">>) {
    if (!selectedId.value) return;
    update(selectedId.value, patch);
}

const toaster = useDesignToaster();
const lastRemoved = ref<Marker>();
let undoTimer: ReturnType<typeof setTimeout> | undefined;

function onRemove() {
    const marker = selectedMarker.value;
    if (!marker) return;
    remove(marker.id);
    selectedId.value = undefined;
    lastRemoved.value = marker;
    clearTimeout(undoTimer);
    undoTimer = setTimeout(() => (lastRemoved.value = undefined), 8000);
}

function undoRemove() {
    if (!lastRemoved.value) return;
    restore(lastRemoved.value);
    selectedId.value = lastRemoved.value.id;
    lastRemoved.value = undefined;
    clearTimeout(undoTimer);
}

async function onSave() {
    await save();
    toaster.create({
        title: t("markers.saved"),
        description: t("markers.savedMock"),
        type: "success",
    });
}

// ---- Keyboard shortcuts (ignored while typing in a field) -------------------
function isTyping(target: EventTarget | null) {
    if (!(target instanceof HTMLElement)) return false;
    const tag = target.tagName;
    return tag === "INPUT" || tag === "TEXTAREA" || target.isContentEditable;
}

useEventListener(window, "keydown", (event: KeyboardEvent) => {
    if (isTyping(event.target)) return;
    const el = videoElement.value;
    switch (event.key) {
        case " ":
            if (!el) return;
            event.preventDefault();
            el.paused ? el.play() : el.pause();
            break;
        case "ArrowRight":
            if (el) el.currentTime += 1;
            break;
        case "ArrowLeft":
            if (el) el.currentTime -= 1;
            break;
        case "m":
        case "M":
            addMarker();
            break;
    }
});
</script>

<template>
    <div
        class="mx-auto flex h-[calc(100vh-var(--header-height,3.5rem))] w-full max-w-[1700px] flex-col gap-3 p-4"
    >
        <header
            class="flex shrink-0 flex-wrap items-center justify-between gap-3"
        >
            <div>
                <NuxtLink
                    to="/markers/"
                    class="text-text-hint hover:text-text-default text-caption-1 inline-flex items-center gap-1"
                >
                    <Icon name="tabler:chevron-left" class="size-3.5" />
                    {{ t("markers.index.title") }}
                </NuxtLink>
                <h1 class="text-heading-3 text-text-default">
                    {{ t("markers.editor.pageTitle") }}
                    <span class="text-text-muted">{{ vxId }}</span>
                </h1>
            </div>
            <div class="flex items-center gap-3">
                <span
                    class="text-text-hint text-caption-1 inline-flex items-center gap-1"
                >
                    <Icon name="tabler:device-floppy" class="size-3.5" />
                    {{ t("markers.savedLocally") }}
                </span>
                <DesignButton
                    icon="tabler:cloud-upload"
                    :loading="saving"
                    @click="onSave"
                >
                    {{ t("markers.save") }}
                </DesignButton>
            </div>
        </header>

        <!-- Workspace: video + timeline (left), list + editor (right) -->
        <div
            class="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-[1fr_380px]"
        >
            <!-- Left column -->
            <div class="flex min-h-0 flex-col gap-3">
                <div
                    class="bg-surface-default relative aspect-video max-h-full w-full shrink-0 overflow-hidden rounded-xl shadow-xl"
                >
                    <video
                        v-if="videoStatus === 'success' && videoUrl"
                        ref="videoElement"
                        :src="videoUrl"
                        controls
                        class="size-full bg-black object-contain"
                    />
                    <DesignSkeleton
                        v-else-if="videoStatus === 'pending'"
                        class="size-full"
                    />
                    <div
                        v-else
                        class="text-text-hint flex size-full flex-col items-center justify-center gap-2 text-sm"
                    >
                        <Icon name="tabler:video-off" class="size-8" />
                        {{ t("markers.previewUnavailable") }}
                    </div>

                    <!-- On-video overlay of markers active at the playhead -->
                    <div
                        v-if="activeMarkers.length"
                        class="pointer-events-none absolute top-4 left-4 flex flex-col gap-1.5"
                    >
                        <div
                            v-for="m in activeMarkers"
                            :key="m.id"
                            class="flex items-center gap-2 rounded-md bg-black/70 px-3 py-1.5 text-sm text-white backdrop-blur"
                        >
                            <Icon
                                :name="markerTypeMeta(m.type).icon"
                                class="size-4 shrink-0"
                                :class="markerTypeMeta(m.type).iconColor"
                            />
                            <span class="font-medium">{{
                                m.label || t(`markers.types.${m.type}`)
                            }}</span>
                        </div>
                    </div>
                </div>

                <!-- Controls (undo notice folds in on the right when present) -->
                <div class="flex shrink-0 items-center gap-3">
                    <span class="text-text-default font-medium tabular-nums">
                        {{ formatTime(currentTime) }}
                    </span>
                    <DesignTooltip :content="t('markers.addHint')">
                        <DesignButton icon="tabler:plus" @click="addMarker">
                            {{ t("markers.addAtPlayhead") }}
                        </DesignButton>
                    </DesignTooltip>
                    <div
                        v-if="lastRemoved"
                        class="ml-auto flex items-center gap-2"
                    >
                        <span class="text-text-muted text-sm">
                            {{
                                t("markers.removed", {
                                    label:
                                        lastRemoved.label ||
                                        t(`markers.types.${lastRemoved.type}`),
                                })
                            }}
                        </span>
                        <DesignButton
                            size="small"
                            variant="tertiary"
                            icon="tabler:arrow-back-up"
                            @click="undoRemove"
                        >
                            {{ t("markers.undo") }}
                        </DesignButton>
                    </div>
                </div>

                <MarkersTimeline
                    class="shrink-0"
                    :markers="visibleMarkers"
                    :duration="effectiveDuration"
                    :current="currentTime"
                    :selected-id="selectedId"
                    @select="(id) => (selectedId = id)"
                    @seek="seek"
                />
            </div>

            <!-- Right column: marker list (scrolls) on top, editor below -->
            <div class="flex min-h-0 flex-col gap-4">
                <div
                    class="border-border-1 bg-surface-default flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border"
                >
                    <div
                        class="border-border-1 text-text-muted flex shrink-0 items-center justify-between border-b px-4 py-2 text-sm"
                    >
                        <span>{{ t("markers.list.title") }}</span>
                        <span class="tabular-nums">{{
                            visibleMarkers.length
                        }}</span>
                    </div>
                    <div
                        v-if="visibleMarkers.length"
                        class="flex flex-col gap-0.5 overflow-y-auto p-2"
                    >
                        <MarkersListItem
                            v-for="marker in visibleMarkers"
                            :key="marker.id"
                            :marker="marker"
                            :selected="marker.id === selectedId"
                            :current-time="currentTime"
                            @select="
                                selectedId = marker.id;
                                seek(marker.start);
                            "
                        />
                    </div>
                    <div
                        v-else
                        class="text-text-hint flex flex-1 items-center justify-center p-8 text-center text-sm"
                    >
                        {{ t("markers.list.empty") }}
                    </div>
                </div>

                <div class="shrink-0">
                    <MarkersEditor
                        :marker="selectedMarker"
                        :current-time="currentTime"
                        @update="onUpdate"
                        @remove="onRemove"
                        @seek="seek"
                    />
                </div>
            </div>
        </div>
    </div>
</template>
