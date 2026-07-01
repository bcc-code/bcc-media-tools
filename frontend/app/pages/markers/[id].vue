<script setup lang="ts">
import { LayoutGroup, motion } from "motion-v";
import { onBeforeRouteLeave } from "vue-router";
import { formatMarkerTime, markerTypeMeta, sortMarkers } from "~/utils/markers";
import type { Marker, MarkerType } from "~/utils/markers";

// Seconds the arrow keys jump the playhead.
const SEEK_STEP_SECONDS = 5;

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
const { markers, dirty, add, update, remove, restore, save, saving } =
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
// When previewing a marker's range, stop playback once we reach its out-point.
const previewEnd = ref<number | null>(null);
useEventListener(videoElement, "timeupdate", () => {
    currentTime.value = videoElement.value?.currentTime ?? 0;
    if (previewEnd.value !== null && currentTime.value >= previewEnd.value) {
        videoElement.value?.pause();
        previewEnd.value = null;
    }
});

function previewRange(start: number, end: number) {
    if (!videoElement.value) return;
    videoElement.value.currentTime = start;
    previewEnd.value = end;
    videoElement.value.play();
}

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
// Remembers the type of the last marker added/edited so the next one defaults
// to it (persisted across sessions).
const lastMarkerType = useLocalStorage<MarkerType>(
    "markers-last-type",
    "name-super",
);

const editor = useTemplateRef("editor");
function addMarker() {
    const start = Math.round(currentTime.value);
    const end = Math.min(start + 5, Math.round(effectiveDuration.value));
    const marker = add({ start, end, type: lastMarkerType.value });
    selectedId.value = marker.id;
    nextTick(() => editor.value?.focusLabel());
}

function onUpdate(patch: Partial<Omit<Marker, "id">>) {
    if (!selectedId.value) return;
    if (patch.type) lastMarkerType.value = patch.type;
    update(selectedId.value, patch);
}

// Select the previous/next marker (by time) and seek to it.
function selectAdjacent(direction: 1 | -1) {
    const list = visibleMarkers.value;
    if (!list.length) return;
    const index = list.findIndex((m) => m.id === selectedId.value);
    const next =
        index === -1
            ? direction === 1
                ? 0
                : list.length - 1
            : Math.max(0, Math.min(list.length - 1, index + direction));
    const marker = list[next];
    if (marker) {
        selectedId.value = marker.id;
        seek(marker.start);
    }
}

// Set the selected marker's In/Out to the playhead (clamped so it can't invert).
function setSelectedBound(which: "start" | "end") {
    const marker = selectedMarker.value;
    if (!marker) return;
    const now = Math.round(currentTime.value);
    if (which === "start")
        update(marker.id, { start: Math.min(now, marker.end) });
    else update(marker.id, { end: Math.max(now, marker.start) });
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

const showShortcuts = ref(false);
const shortcuts = computed(() => [
    { keys: ["Space"], label: t("markers.shortcuts.playPause") },
    { keys: ["←", "→"], label: t("markers.shortcuts.seek") },
    { keys: ["↑", "↓"], label: t("markers.shortcuts.prevNext") },
    { keys: ["M"], label: t("markers.shortcuts.add") },
    { keys: ["I", "O"], label: t("markers.shortcuts.setInOut") },
    { keys: ["Del"], label: t("markers.shortcuts.remove") },
    { keys: ["?"], label: t("markers.shortcuts.help") },
]);

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
            if (el) el.currentTime += SEEK_STEP_SECONDS;
            break;
        case "ArrowLeft":
            if (el) el.currentTime -= SEEK_STEP_SECONDS;
            break;
        case "ArrowUp":
            event.preventDefault();
            selectAdjacent(-1);
            break;
        case "ArrowDown":
            event.preventDefault();
            selectAdjacent(1);
            break;
        case "m":
        case "M":
            addMarker();
            break;
        case "i":
        case "I":
            setSelectedBound("start");
            break;
        case "o":
        case "O":
            setSelectedBound("end");
            break;
        case "Delete":
        case "Backspace":
            onRemove();
            break;
        case "?":
            showShortcuts.value = !showShortcuts.value;
            break;
    }
});

// Warn before losing unsaved local changes.
onBeforeRouteLeave(() => {
    if (dirty.value && !window.confirm(t("markers.leaveConfirm"))) return false;
});
useEventListener(window, "beforeunload", (event: BeforeUnloadEvent) => {
    if (dirty.value) event.preventDefault();
});
</script>

<template>
    <div
        class="mx-auto flex h-[calc(100vh-var(--header-height,3.5rem))] w-full max-w-[1600px] flex-col gap-3 p-4"
    >
        <header
            class="flex shrink-0 flex-wrap items-center justify-between gap-3"
        >
            <h1 class="text-heading-3 text-text-default">
                {{ t("markers.editor.pageTitle") }}
                <span class="text-text-muted">{{ vxId }}</span>
            </h1>

            <div class="flex items-center gap-3">
                <span
                    class="text-caption-1 inline-flex items-center gap-1"
                    :class="dirty ? 'text-semantic-warning' : 'text-text-hint'"
                >
                    <Icon
                        :name="dirty ? 'tabler:point-filled' : 'tabler:check'"
                        class="size-3.5"
                    />
                    {{ dirty ? t("markers.unsaved") : t("markers.allSaved") }}
                </span>
                <DesignTooltip :content="t('markers.shortcuts.title')">
                    <button
                        type="button"
                        class="ds-focus-ring text-text-muted hover:bg-surface-indent hover:text-text-default flex items-center justify-center rounded-lg p-2 transition-colors"
                        @click="showShortcuts = true"
                    >
                        <Icon name="tabler:keyboard" class="size-4" />
                    </button>
                </DesignTooltip>
                <DesignButton
                    icon="tabler:cloud-upload"
                    :loading="saving"
                    :disabled="!dirty"
                    @click="onSave"
                >
                    {{ t("markers.save") }}
                </DesignButton>
            </div>
        </header>

        <div
            class="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-[1fr_380px]"
        >
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

                <div class="flex shrink-0 items-center gap-3">
                    <span class="text-text-default font-medium tabular-nums">
                        {{ formatMarkerTime(currentTime) }}
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

            <LayoutGroup>
                <div class="flex min-h-0 flex-col gap-4">
                    <motion.div
                        layout="position"
                        :transition="{ duration: 1, ease: [0.16, 1, 0.3, 1] }"
                        class="shrink-0"
                    >
                        <MarkersEditor
                            ref="editor"
                            :marker="selectedMarker"
                            :current-time="currentTime"
                            @update="onUpdate"
                            @remove="onRemove"
                            @seek="seek"
                            @preview="previewRange"
                        />
                    </motion.div>

                    <motion.div
                        layout="position"
                        :transition="{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }"
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
                            class="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto p-2"
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
                    </motion.div>
                </div>
            </LayoutGroup>
        </div>

        <DesignDialog
            v-model:open="showShortcuts"
            :title="t('markers.shortcuts.title')"
        >
            <div class="flex flex-col gap-2">
                <div
                    v-for="(shortcut, i) in shortcuts"
                    :key="i"
                    class="flex items-center justify-between gap-4"
                >
                    <span class="text-body-3 text-text-default">
                        {{ shortcut.label }}
                    </span>
                    <span class="flex shrink-0 items-center gap-1">
                        <kbd
                            v-for="key in shortcut.keys"
                            :key="key"
                            class="border-border-1 bg-surface-indent text-text-muted text-caption-1 min-w-6 rounded border px-1.5 py-0.5 text-center tabular-nums"
                        >
                            {{ key }}
                        </kbd>
                    </span>
                </div>
            </div>
        </DesignDialog>
    </div>
</template>
