<script setup lang="ts">
import type {
    EditorialSession,
    EditorialMarker,
    PlayoutEvent,
    GetRecordingWindowResponse,
} from "~~/src/gen/api/v1/api_pb";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate, timestampFromDate } from "@bufbuild/protobuf/wkt";

const route = useRoute("editorial-id");
const sessionId = computed(() => route.params.id as string);

const { t, te } = useI18n();
const api = useAPI();
const perms = usePermissions();
const toaster = useToast();

const canEdit = perms.canEditEditorial;

useHead({ title: t("tools.editorial.title") });

const analytics = useAnalytics();
onMounted(() =>
    analytics.page({
        id: "editorial_session",
        title: "editorial",
        meta: { id: sessionId.value },
    }),
);

const TYPE_OPTIONS = [
    "vitnesbyrd",
    "sang",
    "allsang",
    "tale",
    "bønn",
    "tydning",
    "programleder",
    "video",
    "intervju",
    "annet",
];

// Translated type label; falls back to the raw value for legacy types.
function typeLabel(type: string): string {
    const key = `editorial.types.${type}`;
    if (te(key)) return t(key);
    return type ? type.charAt(0).toUpperCase() + type.slice(1) : type;
}

const typeItems = computed(() =>
    TYPE_OPTIONS.map((value) => ({ label: typeLabel(value), value })),
);

// A single editable row. Start/End are kept as "HH:MM:SS" strings so text
// editing is natural; they're parsed to milliseconds only at save/preview.
interface Row {
    key: string;
    id: string;
    name: string;
    contributors: string;
    bibleVerses: string;
    comment: string;
    type: string;
    start: string;
    end: string;
    publishBmm: boolean;
    publishBcc: boolean;
    source: string;
}

const session = ref<EditorialSession>();
const title = ref("");
const rows = ref<Row[]>([]);
const loading = ref(true);
const notFound = ref(false);

const mode = ref<"simple" | "edit">("simple");
const effectiveMode = computed(() => (canEdit.value ? mode.value : "simple"));

// Fixed-width mode control, so switching modes doesn't shift the layout.
const modeItems = computed(() => [
    { label: t("editorial.viewSimple"), value: "simple" },
    { label: t("editorial.viewEdit"), value: "edit" },
]);
const modeModel = computed<string>({
    get: () => mode.value,
    set: (v) => (mode.value = v === "edit" ? "edit" : "simple"),
});

const dirty = ref(false);
let hydrated = false;

function formatMs(ms: number): string {
    const total = Math.max(0, Math.floor(ms / 1000));
    const h = Math.floor(total / 3600);
    const m = Math.floor((total % 3600) / 60);
    const s = total % 60;
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${pad(h)}:${pad(m)}:${pad(s)}`;
}

const dateTimeFormatter = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Europe/Oslo",
    dateStyle: "medium",
    timeStyle: "short",
});
function formatDateTime(ts?: Timestamp): string {
    if (!ts) return "";
    return dateTimeFormatter.format(timestampDate(ts));
}

// Parses "HH:MM:SS(.mmm)", "MM:SS" or "SS" into milliseconds; invalid → 0.
function parseTc(tc: string): number {
    const parts = tc
        .trim()
        .split(":")
        .map((p) => Number(p));
    if (parts.some((n) => Number.isNaN(n))) return 0;
    let seconds = 0;
    for (const p of parts) seconds = seconds * 60 + p;
    return Math.max(0, Math.round(seconds * 1000));
}

let rowSeq = 0;
const nextRowKey = () => `r${++rowSeq}`;

function toRow(m: EditorialMarker): Row {
    return {
        key: nextRowKey(),
        id: m.id,
        name: m.name,
        contributors: m.contributors,
        bibleVerses: m.bibleVerses,
        comment: m.comment,
        type: m.type,
        start: formatMs(Number(m.startMs)),
        end: formatMs(Number(m.endMs)),
        publishBmm: m.publishBmm,
        publishBcc: m.publishBcc,
        source: m.source || "manual",
    };
}

function durationOf(row: Row): string {
    return formatMs(parseTc(row.end) - parseTc(row.start));
}

const previewUrl = ref<string>();
const videoEl = useTemplateRef<HTMLVideoElement>("videoEl");

function preview(row: Row) {
    const el = videoEl.value;
    if (!el) return;
    el.currentTime = parseTc(row.start) / 1000;
    void el.play();
}

// Highlight the marker whose [start, end) range contains the playhead.
const currentMs = ref(0);
function onTimeUpdate(e: Event) {
    currentMs.value = (e.target as HTMLVideoElement).currentTime * 1000;
}
const activeIndex = computed(() =>
    rows.value.findIndex((r) => {
        const start = parseTc(r.start);
        const end = parseTc(r.end);
        return end > start && currentMs.value >= start && currentMs.value < end;
    }),
);
const activeMarker = computed(() =>
    activeIndex.value >= 0 ? rows.value[activeIndex.value] : undefined,
);
// Playhead position within the active marker (ms). Reading follows playback;
// setting (dragging the slider) scrubs the video within the marker.
const scrubMs = computed<number>({
    get() {
        const m = activeMarker.value;
        if (!m) return 0;
        const start = parseTc(m.start);
        const end = parseTc(m.end);
        return Math.min(end, Math.max(start, currentMs.value));
    },
    set(v) {
        const el = videoEl.value;
        if (!el) return;
        el.currentTime = v / 1000;
        currentMs.value = v;
    },
});

async function load() {
    loading.value = true;
    notFound.value = false;
    hydrated = false;
    try {
        const s = await api.getEditorialSession({ id: sessionId.value });
        session.value = s;
        title.value = s.title;
        rows.value = s.markers.map(toRow);
        previewUrl.value = s.previewUrl || undefined;
    } catch {
        notFound.value = true;
    } finally {
        loading.value = false;
        await nextTick();
        hydrated = true;
    }
}
onMounted(load);

watch(
    [title, rows],
    () => {
        if (hydrated) dirty.value = true;
    },
    { deep: true },
);

// ── Auto-save status ──────────────────────────────────────
// Simple-mode edits persist immediately (no Save button), so surface an inline
// "Saving…/Saved" indicator. `tracked` wraps each write; concurrent writes are
// counted so the indicator only settles once the last one lands.
type SaveStatus = "idle" | "saving" | "saved" | "error";
const saveStatus = ref<SaveStatus>("idle");
let inFlight = 0;
let inFlightError = false;
let saveResetTimer: ReturnType<typeof setTimeout> | undefined;

async function tracked<T>(fn: () => Promise<T>): Promise<T> {
    if (saveResetTimer) {
        clearTimeout(saveResetTimer);
        saveResetTimer = undefined;
    }
    inFlight++;
    saveStatus.value = "saving";
    try {
        return await fn();
    } catch (e) {
        inFlightError = true;
        throw e;
    } finally {
        inFlight--;
        if (inFlight === 0) {
            const errored = inFlightError;
            inFlightError = false;
            // These RPCs return Void; reflect the server's touch of updated_at
            // locally so "Last updated" tracks simple-mode edits without a reload.
            if (!errored && session.value) {
                session.value.updatedAt = timestampFromDate(new Date());
            }
            saveStatus.value = errored ? "error" : "saved";
            saveResetTimer = setTimeout(
                () => {
                    saveStatus.value = "idle";
                    saveResetTimer = undefined;
                },
                errored ? 4000 : 2000,
            );
        }
    }
}

// ── Publish toggle ────────────────────────────────────────
async function onPublishToggle(
    row: Row,
    target: "bmm" | "bcc",
    value: boolean,
) {
    if (target === "bmm") row.publishBmm = value;
    else row.publishBcc = value;
    // In edit mode the change is persisted on Save. In simple mode there is no
    // Save button, so persist immediately (both flags, from the row's state).
    if (effectiveMode.value === "edit") return;
    try {
        await tracked(() =>
            api.setEditorialPublish({
                sessionId: sessionId.value,
                markerId: row.id,
                publishBmm: row.publishBmm,
                publishBcc: row.publishBcc,
            }),
        );
    } catch {
        if (target === "bmm") row.publishBmm = !value;
        else row.publishBcc = !value;
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    }
}

// ── Row title/name (editable in both views, edit rights required) ──
const nameBeforeEdit = ref("");

// Edit mode persists on the batch Save; simple mode writes immediately.
async function persistName(row: Row) {
    if (effectiveMode.value === "edit") return;
    if (!row.id || row.name === nameBeforeEdit.value) return;
    try {
        await tracked(() =>
            api.setEditorialName({
                sessionId: sessionId.value,
                markerId: row.id,
                name: row.name,
            }),
        );
    } catch {
        row.name = nameBeforeEdit.value;
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    }
}

// ── Comment (editable in both views) ──────────────────────
const commentBeforeEdit = ref("");

// Edit mode persists on the batch Save; simple mode writes immediately.
async function persistComment(row: Row) {
    if (effectiveMode.value === "edit") return;
    if (!row.id || row.comment === commentBeforeEdit.value) return;
    try {
        await tracked(() =>
            api.setEditorialComment({
                sessionId: sessionId.value,
                markerId: row.id,
                comment: row.comment,
            }),
        );
    } catch {
        row.comment = commentBeforeEdit.value;
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    }
}

// ── Edit-mode mutations ───────────────────────────────────
function addRow() {
    rows.value.push({
        key: nextRowKey(),
        id: "",
        name: "",
        contributors: "",
        bibleVerses: "",
        comment: "",
        type: "",
        start: "00:00:00",
        end: "00:00:00",
        publishBmm: false,
        publishBcc: false,
        source: "manual",
    });
}

function removeRow(i: number) {
    const row = rows.value[i];
    if (row) selectedKeys.value.delete(row.key);
    rows.value.splice(i, 1);
}

// ── Bulk selection ────────────────────────────────────────
// Imports can add twenty-odd rows at once; removing the unwanted ones one
// trash-click at a time is the slow part of the review
const selectedKeys = ref(new Set<string>());
const selectedCount = computed(() => selectedKeys.value.size);
const allSelected = computed(
    () => rows.value.length > 0 && selectedCount.value === rows.value.length,
);

function isSelected(row: Row) {
    return selectedKeys.value.has(row.key);
}

let rangeAnchor: number | null = null;
let shiftHeld = false;
function noteModifier(e: MouseEvent) {
    shiftHeld = e.shiftKey;
    if (!e.shiftKey) return;
    e.preventDefault();
    window.getSelection()?.removeAllRanges();
}

function setSelected(row: Row, on: boolean) {
    const index = rows.value.findIndex((r) => r.key === row.key);
    const next = new Set(selectedKeys.value);
    if (shiftHeld && rangeAnchor !== null && index >= 0) {
        const from = Math.min(rangeAnchor, index);
        const to = Math.max(rangeAnchor, index);
        for (let i = from; i <= to; i++) {
            const key = rows.value[i]?.key;
            if (!key) continue;
            if (on) next.add(key);
            else next.delete(key);
        }
    } else {
        if (on) next.add(row.key);
        else next.delete(row.key);
        // The anchor only moves on a plain click, so the range can be widened or narrowed with repeated shift-clicks.
        rangeAnchor = index;
    }
    selectedKeys.value = next;
    shiftHeld = false;
}
function toggleAll(on: boolean) {
    selectedKeys.value = on ? new Set(rows.value.map((r) => r.key)) : new Set();
    rangeAnchor = null;
}
function removeSelected() {
    rows.value = rows.value.filter((r) => !selectedKeys.value.has(r.key));
    selectedKeys.value = new Set();
    rangeAnchor = null;
    dirty.value = true;
}

// Drag-to-reorder. Only the grip handle is draggable so the row's inputs stay
// selectable; the whole row is used as the drag image for clear feedback.
const dragIndex = ref<number | null>(null);
const dragOverIndex = ref<number | null>(null);

function onDragStart(i: number, e: DragEvent) {
    dragIndex.value = i;
    if (!e.dataTransfer) return;
    e.dataTransfer.effectAllowed = "move";
    // Firefox only starts a drag once data is set.
    e.dataTransfer.setData("text/plain", String(i));
    const tr = (e.target as HTMLElement).closest("tr");
    if (tr) e.dataTransfer.setDragImage(tr, 24, 16);
}

function onDragOver(i: number) {
    if (dragIndex.value !== null) dragOverIndex.value = i;
}

function onDrop(i: number) {
    const from = dragIndex.value;
    onDragEnd();
    if (from === null || from === i) return;
    const [item] = rows.value.splice(from, 1);
    rows.value.splice(i, 0, item!);
}

function onDragEnd() {
    dragIndex.value = null;
    dragOverIndex.value = null;
}

// ── Backend actions ───────────────────────────────────────
const importing = ref(false);
const importingPlayout = ref(false);
const saving = ref(false);
const mutationInProgress = computed(
    () => importing.value || importingPlayout.value || saving.value,
);
const deleteOpen = ref(false);

// ── Import merge ──────────────────────────────────────────
// A second import is usually a retry after correcting the window, so appending
// would double the table. Ask, but only when there's something to replace.
const importMergeOpen = ref(false);
const pendingImport = ref<Row[]>([]);
const importedRowCount = computed(
    () => rows.value.filter((r) => r.source === "import").length,
);

function applyImport(markers: EditorialMarker[]) {
    const incoming = markers.map(toRow);
    if (importedRowCount.value === 0) {
        commitImport(incoming, false);
        return;
    }
    pendingImport.value = incoming;
    importMergeOpen.value = true;
}

function commitImport(incoming: Row[], replace: boolean) {
    if (replace) {
        rows.value = rows.value.filter((r) => r.source !== "import");
        selectedKeys.value = new Set();
    }
    rows.value.push(...incoming);
    dirty.value = true;
    toaster.create({
        title: t("editorial.importedCount", { n: incoming.length }),
        type: "success",
    });
}

function resolveImportMerge(replace: boolean) {
    importMergeOpen.value = false;
    commitImport(pendingImport.value, replace);
    pendingImport.value = [];
}

const playoutPickerOpen = ref(false);
const playoutEvents = ref<PlayoutEvent[]>([]);
const loadingPlayoutEvents = ref(false);
const playoutSearch = ref("");
const filteredPlayoutEvents = computed(() => {
    const q = playoutSearch.value.trim().toLowerCase();
    if (!q) return playoutEvents.value;
    return playoutEvents.value.filter((e) => e.name.toLowerCase().includes(q));
});

// ── Recording window ──────────────────────────────────────
// A Playout event covers a whole conference, so the import needs this
// recording's span. The server derives it from the file name — a convention,
// not a guarantee — so it's shown: a wrong window imports the wrong meeting.
const recWindow = ref<GetRecordingWindowResponse>();
const loadingRecWindow = ref(false);
const recAdjusting = ref(false);
const recDate = ref("");
const recStartTime = ref("");
const recEndTime = ref("");

// No usable duration, so an end can't be inferred and must be entered.
const recNeedsEnd = computed(() => (recWindow.value?.durationMs ?? 0n) === 0n);

const osloTime = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Europe/Oslo",
    timeStyle: "short",
});

// Europe/Oslo's offset at an instant: format it there, read it back as UTC,
// and the gap is the offset. The inputs are wall-clock; the API is UTC.
function osloOffsetMs(at: Date): number {
    const parts = new Intl.DateTimeFormat("en-GB", {
        timeZone: "Europe/Oslo",
        hourCycle: "h23",
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
    }).formatToParts(at);
    const n = (type: string) =>
        Number(parts.find((p) => p.type === type)?.value ?? 0);
    return (
        Date.UTC(
            n("year"),
            n("month") - 1,
            n("day"),
            n("hour"),
            n("minute"),
            n("second"),
        ) - at.getTime()
    );
}

// Reads "YYYY-MM-DD" + "HH:MM" as Oslo wall-clock time. Resolved twice: the
// first pass can land on the wrong side of a DST change.
function osloToDate(date: string, time: string): Date | undefined {
    const d = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date.trim());
    const t = /^(\d{1,2}):(\d{2})$/.exec(time.trim());
    if (!d || !t) return undefined;
    const naive = Date.UTC(+d[1]!, +d[2]! - 1, +d[3]!, +t[1]!, +t[2]!);
    const first = new Date(naive - osloOffsetMs(new Date(naive)));
    return new Date(naive - osloOffsetMs(first));
}

function formatDuration(ms: number): string {
    const minutes = Math.round(ms / 60000);
    const h = Math.floor(minutes / 60);
    const m = minutes % 60;
    return h > 0 ? `${h}h${String(m).padStart(2, "0")}m` : `${m}m`;
}

// "22 Sept 2026, 14:45–16:38 (1h53m)"
const recWindowSummary = computed(() => {
    const w = recWindow.value;
    if (!w?.start || !w?.end) return "";
    const start = timestampDate(w.start);
    const end = timestampDate(w.end);
    return `${formatDateTime(w.start)}-${osloTime.format(end)} (${formatDuration(
        end.getTime() - start.getTime(),
    )})`;
});

// Until the editor sets an end themselves, it follows the start so moving the
// start doesn't silently shorten the window.
const recEndEdited = ref(false);
let syncingEnd = false;
function setEndTime(value: string) {
    syncingEnd = true;
    recEndTime.value = value;
    syncingEnd = false;
}
watch(
    recEndTime,
    () => {
        if (!syncingEnd) recEndEdited.value = true;
    },
    { flush: "sync" },
);
watch([recDate, recStartTime], () => {
    const duration = Number(recWindow.value?.durationMs ?? 0n);
    if (recEndEdited.value || !duration) return;
    const start = osloToDate(recDate.value, recStartTime.value);
    if (start)
        setEndTime(osloTime.format(new Date(start.getTime() + duration)));
});

// Splits a window into the inputs, in Oslo time to match what's shown.
function fillRecInputs(w?: GetRecordingWindowResponse) {
    const start = w?.start ? timestampDate(w.start) : undefined;
    const end = w?.end ? timestampDate(w.end) : undefined;
    const osloDate = (d: Date) =>
        new Intl.DateTimeFormat("en-CA", {
            timeZone: "Europe/Oslo",
        }).format(d);
    recDate.value = start ? osloDate(start) : "";
    recStartTime.value = start ? osloTime.format(start) : "";
    setEndTime(end ? osloTime.format(end) : "");
    // A stored end that isn't just start + duration was chosen deliberately;
    // don't let the follow-the-start rule overwrite it.
    const duration = Number(w?.durationMs ?? 0n);
    recEndEdited.value = Boolean(
        start &&
        end &&
        duration &&
        Math.abs(end.getTime() - start.getTime() - duration) > 60_000,
    );
}

async function loadRecordingWindow() {
    loadingRecWindow.value = true;
    recAdjusting.value = false;
    recWindow.value = undefined;
    try {
        const w = await api.getRecordingWindow({ sessionId: sessionId.value });
        recWindow.value = w;
        fillRecInputs(w);
        // Nothing to confirm without a window — ask straight away.
        if (!w.start) recAdjusting.value = true;
    } catch (e) {
        toaster.create({
            title: t("editorial.importFailed"),
            description: e instanceof Error ? e.message : undefined,
            type: "error",
        });
        recAdjusting.value = true;
    } finally {
        loadingRecWindow.value = false;
    }
}

// Secondary/destructive actions live in the overflow menu; Save stays primary.
const menuItems = computed(() => [
    ...(effectiveMode.value === "edit"
        ? [
              {
                  value: "import",
                  label: t("editorial.importVidispine"),
                  icon: "tabler:download",
                  disabled: mutationInProgress.value,
              },
              {
                  value: "import-playout",
                  label: t("editorial.importPlayout"),
                  icon: "tabler:download",
                  disabled: mutationInProgress.value,
              },
          ]
        : []),
    {
        value: "delete",
        label: t("editorial.delete"),
        icon: "tabler:trash",
        intent: "danger" as const,
    },
]);
function onMenuSelect(value: string) {
    if (value === "delete") deleteOpen.value = true;
    else if (value === "import") void importMarkers();
    else if (value === "import-playout") void openPlayoutPicker();
}

async function openPlayoutPicker() {
    if (mutationInProgress.value) return;
    playoutSearch.value = "";
    playoutPickerOpen.value = true;
    loadingPlayoutEvents.value = true;
    void loadRecordingWindow();
    try {
        const res = await api.listPlayoutEvents({});
        playoutEvents.value = res.events;
    } catch {
        toaster.create({ title: t("editorial.importFailed"), type: "error" });
        playoutPickerOpen.value = false;
    } finally {
        loadingPlayoutEvents.value = false;
    }
}

function selectPlayoutEvent(eventId: string) {
    // Only send a window if the inputs were opened; otherwise the server
    // re-derives it, keeping the untouched case at two clicks.
    let start: Date | undefined;
    let end: Date | undefined;
    if (recAdjusting.value) {
        start = osloToDate(recDate.value, recStartTime.value);
        if (!start) {
            toaster.create({
                title: t("editorial.recordingWindowInvalid"),
                type: "error",
            });
            return;
        }
        if (recEndTime.value.trim()) {
            end = osloToDate(recDate.value, recEndTime.value);
            if (!end) {
                toaster.create({
                    title: t("editorial.recordingWindowInvalid"),
                    type: "error",
                });
                return;
            }
            // An earlier end means the recording ran past midnight.
            if (end <= start) end = new Date(end.getTime() + 86400000);
        } else if (recNeedsEnd.value) {
            toaster.create({
                title: t("editorial.recordingWindowInvalid"),
                type: "error",
            });
            return;
        }
    }
    playoutPickerOpen.value = false;
    void importPlayoutMarkers(eventId, start, end);
}

async function importMarkers() {
    if (mutationInProgress.value) return;
    importing.value = true;
    try {
        const res = await api.importEditorialMarkers({ id: sessionId.value });
        applyImport(res.markers);
    } catch {
        toaster.create({ title: t("editorial.importFailed"), type: "error" });
    } finally {
        importing.value = false;
    }
}

async function importPlayoutMarkers(
    eventId: string,
    recordingStart?: Date,
    recordingEnd?: Date,
) {
    if (mutationInProgress.value) return;
    importingPlayout.value = true;
    try {
        const res = await api.importEditorialMarkersFromPlayout({
            id: sessionId.value,
            eventId,
            recordingStart: recordingStart
                ? timestampFromDate(recordingStart)
                : undefined,
            recordingEnd: recordingEnd
                ? timestampFromDate(recordingEnd)
                : undefined,
        });
        applyImport(res.markers);
    } catch (e) {
        // Surface the server's specific reason, a bare "import failed" would
        // leave the editor with no way to act on it.
        toaster.create({
            title: t("editorial.importFailed"),
            description: e instanceof Error ? e.message : undefined,
            type: "error",
        });
    } finally {
        importingPlayout.value = false;
    }
}

async function save() {
    if (mutationInProgress.value) return;
    saving.value = true;
    try {
        const s = await api.saveEditorialSession({
            id: sessionId.value,
            title: title.value,
            markers: rows.value.map((r, i) => ({
                id: r.id,
                sortOrder: i,
                name: r.name,
                contributors: r.contributors,
                bibleVerses: r.bibleVerses,
                comment: r.comment,
                type: r.type,
                startMs: BigInt(parseTc(r.start)),
                endMs: BigInt(parseTc(r.end)),
                publishBmm: r.publishBmm,
                publishBcc: r.publishBcc,
                source: r.source,
            })),
        });
        session.value = s;
        title.value = s.title;
        rows.value = s.markers.map(toRow);
        await nextTick();
        dirty.value = false;
        toaster.create({ title: t("editorial.saved"), type: "success" });
    } catch {
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    } finally {
        saving.value = false;
    }
}

async function remove() {
    try {
        await api.deleteEditorialSession({ id: sessionId.value });
        dirty.value = false;
        deleteOpen.value = false;
        toaster.create({ title: t("editorial.deleted"), type: "success" });
        await navigateTo("/editorial/");
    } catch {
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    }
}

onBeforeRouteLeave(() => {
    if (canEdit.value && dirty.value) {
        return window.confirm(t("editorial.unsavedWarning"));
    }
});
</script>

<template>
    <div v-if="!perms.canUseEditorial.value" class="py-16 text-center">
        <p class="text-body-2 text-text-muted">{{ t("noPermissions") }}</p>
    </div>

    <div v-else class="mx-auto w-full max-w-[1700px] px-4 py-6">
        <NuxtLink
            to="/editorial/"
            class="text-caption-1 text-text-hint hover:text-text-default mb-4 inline-flex items-center gap-1"
        >
            <Icon name="tabler:chevron-left" class="size-4" />
            {{ t("editorial.backToList") }}
        </NuxtLink>

        <div v-if="loading" class="flex flex-col gap-2">
            <DesignSkeleton v-for="i in 6" :key="i" class="h-10 rounded-xl" />
        </div>

        <p
            v-else-if="notFound"
            class="text-body-3 text-text-hint py-16 text-center"
        >
            {{ t("editorial.loadFailed") }}
        </p>

        <template v-else>
            <div class="mb-4 max-w-xl">
                <DesignInput
                    v-if="effectiveMode === 'edit'"
                    v-model="title"
                    :placeholder="session?.VXID"
                />
                <h1 v-else class="text-heading-2 text-text-default truncate">
                    {{ title || session?.VXID }}
                </h1>
                <p
                    v-if="title && title !== session?.VXID"
                    class="text-caption-1 text-text-hint mt-1"
                >
                    {{ session?.VXID }}
                </p>
                <p
                    v-if="session?.updatedAt"
                    class="text-caption-1 text-text-hint mt-1"
                >
                    {{
                        t("editorial.lastUpdated", {
                            date: formatDateTime(session.updatedAt),
                        })
                    }}
                </p>
            </div>

            <div
                v-if="canEdit || saveStatus !== 'idle'"
                class="mb-6 flex items-center gap-3"
            >
                <DesignSegmentGroup
                    v-if="canEdit"
                    v-model="modeModel"
                    :items="modeItems"
                    class="border-border-1 border"
                />
                <div class="ml-auto flex items-center gap-3">
                    <span
                        v-if="saveStatus !== 'idle'"
                        class="text-caption-1 flex items-center gap-1.5"
                        :class="
                            saveStatus === 'error'
                                ? 'text-semantic-error'
                                : 'text-text-hint'
                        "
                    >
                        <template v-if="saveStatus === 'saving'">
                            <Icon
                                name="svg-spinners:ring-resize"
                                class="size-4"
                            />
                            {{ t("editorial.saving") }}
                        </template>
                        <template v-else-if="saveStatus === 'saved'">
                            <Icon
                                name="tabler:check"
                                class="text-semantic-success size-4"
                            />
                            {{ t("editorial.saved") }}
                        </template>
                        <template v-else>
                            <Icon name="tabler:alert-triangle" class="size-4" />
                            {{ t("editorial.saveFailed") }}
                        </template>
                    </span>
                    <template v-if="canEdit">
                        <DesignMenu
                            :items="menuItems"
                            :trigger-label="t('editorial.moreActions')"
                            @select="onMenuSelect"
                        />
                        <DesignButton
                            v-if="effectiveMode === 'edit'"
                            icon="tabler:device-floppy"
                            :loading="saving"
                            :disabled="importing || importingPlayout"
                            @click="save"
                        >
                            {{ t("editorial.save") }}
                        </DesignButton>
                    </template>
                </div>
            </div>

            <div class="grid gap-6 lg:grid-cols-[1fr_500px]">
                <div>
                    <p
                        v-if="rows.length === 0"
                        class="text-body-3 text-text-hint bg-surface-indent rounded-2xl px-4 py-10 text-center"
                    >
                        {{ t("editorial.noMarkers") }}
                    </p>
                    <!--
                        Floats over the page rather than sitting in the flow:
                        in-flow it pushed the table and the preview down every
                        time a row was ticked. Fixed also keeps it reachable
                        without scrolling back up a long import.
                    -->
                    <Transition
                        enter-active-class="transition duration-150 ease-out"
                        enter-from-class="translate-y-3 opacity-0"
                        leave-active-class="transition duration-100 ease-in"
                        leave-to-class="translate-y-3 opacity-0"
                    >
                        <div
                            v-if="
                                rows.length &&
                                effectiveMode === 'edit' &&
                                selectedCount
                            "
                            class="bg-surface-raise gradient-border shadow-resting fixed bottom-6 left-1/2 z-40 flex -translate-x-1/2 items-center gap-4 rounded-2xl py-2.5 pr-2.5 pl-5"
                        >
                            <p class="text-body-3 text-text-default">
                                {{
                                    t("editorial.selectedCount", {
                                        n: selectedCount,
                                    })
                                }}
                            </p>
                            <div class="flex items-center gap-2">
                                <DesignButton
                                    variant="tertiary"
                                    size="small"
                                    @click="toggleAll(false)"
                                >
                                    {{ t("editorial.clearSelection") }}
                                </DesignButton>
                                <DesignButton
                                    variant="primary"
                                    intent="danger"
                                    size="small"
                                    icon="tabler:trash"
                                    @click="removeSelected"
                                >
                                    {{ t("editorial.removeSelected") }}
                                </DesignButton>
                            </div>
                        </div>
                    </Transition>

                    <div v-if="rows.length" class="overflow-x-auto">
                        <table class="w-full border-separate border-spacing-0">
                            <thead
                                class="text-caption-1 text-text-hint text-left"
                            >
                                <tr>
                                    <th
                                        v-if="effectiveMode === 'edit'"
                                        class="border-border-1 w-8 border-b py-2 pl-2 select-none"
                                    >
                                        <DesignCheckbox
                                            :model-value="allSelected"
                                            :aria-label="
                                                t('editorial.selectAll')
                                            "
                                            @update:model-value="toggleAll"
                                        />
                                    </th>
                                    <th
                                        class="border-border-1 w-10 border-b py-2 pl-2"
                                    ></th>
                                    <th
                                        class="border-border-1 border-b py-2 pr-2 pl-3 font-normal"
                                    >
                                        {{ t("editorial.col.title") }}
                                    </th>
                                    <th
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.contributors") }}
                                    </th>
                                    <th
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.bibleVerses") }}
                                    </th>
                                    <th
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.duration") }}
                                    </th>
                                    <th
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.type") }}
                                    </th>
                                    <th
                                        v-if="effectiveMode === 'edit'"
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.start") }}
                                    </th>
                                    <th
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.comment") }}
                                    </th>
                                    <th
                                        class="border-border-1 border-b px-2 py-2 text-center font-normal"
                                    >
                                        {{ t("editorial.col.publishBmm") }}
                                    </th>
                                    <th
                                        class="border-border-1 border-b px-2 py-2 text-center font-normal"
                                    >
                                        {{ t("editorial.col.publishBcc") }}
                                    </th>
                                    <th
                                        v-if="effectiveMode === 'edit'"
                                        class="border-border-1 border-b py-2 pl-2"
                                    ></th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr
                                    v-for="(row, i) in rows"
                                    :key="row.key"
                                    class="[&>td]:border-border-1/50 transition-colors [&>td]:border-b"
                                    :class="[
                                        i === activeIndex
                                            ? 'bg-primary-default/10'
                                            : '',
                                        i === dragIndex ? 'opacity-40' : '',
                                        dragOverIndex === i && dragIndex !== i
                                            ? '[&>td]:border-t-primary-default [&>td]:border-t'
                                            : '',
                                    ]"
                                    @dragover.prevent="onDragOver(i)"
                                    @drop.prevent="onDrop(i)"
                                    @dragend="onDragEnd"
                                >
                                    <td
                                        v-if="effectiveMode === 'edit'"
                                        class="py-2 pl-2 select-none"
                                        @mousedown.capture="noteModifier"
                                    >
                                        <DesignCheckbox
                                            :model-value="isSelected(row)"
                                            :title="
                                                t('editorial.selectRowHint')
                                            "
                                            :aria-label="
                                                t('editorial.selectRow')
                                            "
                                            @update:model-value="
                                                setSelected(row, $event)
                                            "
                                        />
                                    </td>
                                    <td class="py-2 pl-2">
                                        <DesignButton
                                            variant="tertiary"
                                            size="small"
                                            icon="tabler:player-play"
                                            :disabled="!previewUrl"
                                            class="border-border-1 border"
                                            @click="preview(row)"
                                        />
                                    </td>
                                    <td class="py-2 pr-2 pl-3">
                                        <DesignInput
                                            v-if="canEdit"
                                            v-model="row.name"
                                            @focusin="nameBeforeEdit = row.name"
                                            @change="persistName(row)"
                                        />
                                        <span
                                            v-else
                                            class="text-body-3 text-text-default"
                                        >
                                            {{ row.name || "—" }}
                                        </span>
                                    </td>
                                    <td class="px-2 py-2">
                                        <DesignInput
                                            v-if="effectiveMode === 'edit'"
                                            v-model="row.contributors"
                                        />
                                        <span
                                            v-else
                                            class="text-body-3 text-text-muted"
                                        >
                                            {{ row.contributors || "—" }}
                                        </span>
                                    </td>
                                    <td class="px-2 py-2">
                                        <DesignInput
                                            v-if="effectiveMode === 'edit'"
                                            v-model="row.bibleVerses"
                                        />
                                        <span
                                            v-else
                                            class="text-body-3 text-text-muted tabular-nums"
                                        >
                                            {{ row.bibleVerses || "—" }}
                                        </span>
                                    </td>
                                    <td
                                        class="text-body-3 text-text-muted px-2 py-2 tabular-nums"
                                    >
                                        {{ durationOf(row) }}
                                    </td>
                                    <td class="px-2 py-2">
                                        <DesignSelect
                                            v-if="effectiveMode === 'edit'"
                                            v-model="row.type"
                                            :items="typeItems"
                                        />
                                        <DesignBadge v-else-if="row.type">
                                            {{ typeLabel(row.type) }}
                                        </DesignBadge>
                                        <span
                                            v-else
                                            class="text-body-3 text-text-hint"
                                        >
                                            —
                                        </span>
                                    </td>
                                    <td
                                        v-if="effectiveMode === 'edit'"
                                        class="px-2 py-2"
                                    >
                                        <div class="flex items-center gap-1">
                                            <DesignInput v-model="row.start" />
                                            <span class="text-text-hint"
                                                >–</span
                                            >
                                            <DesignInput v-model="row.end" />
                                        </div>
                                    </td>
                                    <td
                                        class="px-2 py-2"
                                        @focusin="
                                            commentBeforeEdit = row.comment
                                        "
                                        @change="persistComment(row)"
                                    >
                                        <DesignInput v-model="row.comment" />
                                    </td>
                                    <td class="px-2 py-2">
                                        <div class="flex justify-center">
                                            <DesignSwitch
                                                :model-value="row.publishBmm"
                                                @update:model-value="
                                                    onPublishToggle(
                                                        row,
                                                        'bmm',
                                                        $event,
                                                    )
                                                "
                                            />
                                        </div>
                                    </td>
                                    <td class="px-2 py-2">
                                        <div class="flex justify-center">
                                            <DesignSwitch
                                                :model-value="row.publishBcc"
                                                @update:model-value="
                                                    onPublishToggle(
                                                        row,
                                                        'bcc',
                                                        $event,
                                                    )
                                                "
                                            />
                                        </div>
                                    </td>
                                    <td
                                        v-if="effectiveMode === 'edit'"
                                        class="py-2 pl-2"
                                    >
                                        <div class="flex items-center gap-0.5">
                                            <button
                                                type="button"
                                                :title="t('editorial.reorder')"
                                                :aria-label="
                                                    t('editorial.reorder')
                                                "
                                                draggable="true"
                                                class="text-text-hint hover:text-text-default hover:bg-surface-indent flex cursor-grab items-center rounded-2xl p-1.5 active:cursor-grabbing"
                                                @dragstart="
                                                    onDragStart(i, $event)
                                                "
                                            >
                                                <Icon
                                                    name="tabler:grip-vertical"
                                                    class="size-4"
                                                />
                                            </button>
                                            <DesignButton
                                                variant="tertiary"
                                                intent="danger"
                                                size="small"
                                                icon="tabler:trash"
                                                @click="removeRow(i)"
                                            />
                                        </div>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>

                    <DesignButton
                        v-if="effectiveMode === 'edit'"
                        variant="tertiary"
                        icon="tabler:plus"
                        class="mt-3"
                        @click="addRow"
                    >
                        {{ t("editorial.addRow") }}
                    </DesignButton>
                </div>

                <div>
                    <div
                        class="gradient-border shadow-resting sticky top-[calc(var(--header-height)+1rem)] overflow-hidden rounded-2xl"
                    >
                        <div
                            class="bg-surface-indent aspect-video overflow-hidden"
                        >
                            <video
                                v-if="previewUrl"
                                ref="videoEl"
                                :src="previewUrl"
                                controls
                                class="h-full w-full"
                                @timeupdate="onTimeUpdate"
                            />
                            <div
                                v-else
                                class="text-text-hint flex h-full items-center justify-center"
                            >
                                <Icon name="tabler:video-off" class="size-8" />
                            </div>
                        </div>
                        <div
                            v-if="activeMarker"
                            class="bg-surface-default border-border-1 border-t px-4 py-4"
                        >
                            <div
                                class="flex items-center justify-between gap-3"
                            >
                                <span
                                    class="text-title-2 text-text-default truncate"
                                >
                                    {{ activeMarker.name || "—" }}
                                </span>
                                <DesignBadge v-if="activeMarker.type">
                                    {{ typeLabel(activeMarker.type) }}
                                </DesignBadge>
                            </div>
                            <p
                                v-if="activeMarker.contributors"
                                class="text-body-3 text-text-muted mt-1 truncate"
                            >
                                {{ activeMarker.contributors }}
                            </p>
                            <DesignSlider
                                v-model="scrubMs"
                                :min="parseTc(activeMarker.start)"
                                :max="parseTc(activeMarker.end)"
                                :step="100"
                                class="mt-3 mb-2 flex-1"
                            />
                            <div
                                class="flex items-center justify-between gap-2"
                            >
                                <span
                                    class="text-caption-1 text-text-muted tabular-nums"
                                >
                                    {{ formatMs(scrubMs) }}
                                </span>
                                <span
                                    class="text-caption-1 text-text-muted tabular-nums"
                                >
                                    {{ activeMarker.end }}
                                </span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </template>
    </div>

    <DesignDialog
        v-model:open="importMergeOpen"
        :title="t('editorial.importMergeTitle')"
        :description="
            t('editorial.importMergeMessage', {
                existing: importedRowCount,
                incoming: pendingImport.length,
            })
        "
    >
        <div class="flex justify-end gap-2">
            <DesignButton
                variant="secondary"
                @click="resolveImportMerge(false)"
            >
                {{ t("editorial.importMergeAdd") }}
            </DesignButton>
            <DesignButton variant="primary" @click="resolveImportMerge(true)">
                {{ t("editorial.importMergeReplace") }}
            </DesignButton>
        </div>
    </DesignDialog>

    <DesignDialog
        v-model:open="deleteOpen"
        :title="t('editorial.deleteConfirmTitle')"
        :description="t('editorial.deleteConfirmMessage')"
    >
        <div class="flex justify-end gap-2">
            <DesignButton variant="secondary" @click="deleteOpen = false">
                {{ t("editorial.cancel") }}
            </DesignButton>
            <DesignButton variant="primary" intent="danger" @click="remove">
                {{ t("editorial.delete") }}
            </DesignButton>
        </div>
    </DesignDialog>

    <DesignDialog
        v-model:open="playoutPickerOpen"
        :title="t('editorial.selectPlayoutEvent')"
        size="lg"
    >
        <div class="flex flex-col gap-3">
            <!--
                The window the import will use. Above the list because picking
                an event imports at once — the only moment to catch a bad one.
            -->
            <div class="flex flex-col gap-2">
                <DesignSkeleton
                    v-if="loadingRecWindow"
                    class="h-16 rounded-xl"
                />

                <template v-else>
                    <div
                        v-if="recWindow?.start"
                        class="bg-surface-raise gradient-border shadow-resting flex items-center justify-between gap-4 rounded-xl px-4 py-2.5"
                    >
                        <div class="flex min-w-0 items-center gap-3">
                            <Icon
                                name="tabler:clock"
                                class="text-text-hint size-5 shrink-0"
                            />
                            <div class="min-w-0">
                                <p
                                    class="text-title-2 text-text-default truncate"
                                >
                                    {{ recWindowSummary }}
                                </p>
                                <p
                                    class="text-caption-1 text-text-hint truncate"
                                >
                                    {{
                                        recWindow.source === "manual"
                                            ? t(
                                                  "editorial.recordingWindowManual",
                                              )
                                            : t(
                                                  "editorial.recordingWindowFromMetadata",
                                              )
                                    }}
                                </p>
                            </div>
                        </div>
                        <DesignButton
                            v-if="!recAdjusting"
                            variant="tertiary"
                            size="small"
                            @click="recAdjusting = true"
                        >
                            {{ t("editorial.recordingWindowAdjust") }}
                        </DesignButton>
                    </div>

                    <DesignBanner
                        v-else
                        variant="warning"
                        icon="tabler:alert-triangle"
                    >
                        <div class="min-w-0">
                            <p>{{ t("editorial.recordingWindowUnknown") }}</p>
                            <p v-if="recWindow?.error" class="opacity-80">
                                {{ recWindow.error }}
                            </p>
                        </div>
                    </DesignBanner>

                    <div
                        v-if="recAdjusting"
                        class="bg-surface-indent flex flex-col gap-3 rounded-xl p-3"
                    >
                        <div class="flex flex-wrap gap-3">
                            <DesignInput
                                v-model="recDate"
                                type="date"
                                :label="t('editorial.recordingDate')"
                                class="min-w-40 grow"
                            />
                            <DesignInput
                                v-model="recStartTime"
                                type="time"
                                :label="t('editorial.recordingStartTime')"
                                class="min-w-28 grow"
                            />
                            <DesignInput
                                v-model="recEndTime"
                                type="time"
                                :label="t('editorial.recordingEndTime')"
                                class="min-w-28 grow"
                            />
                        </div>
                        <p class="text-caption-1 text-text-hint">
                            {{ t("editorial.recordingWindowHint") }}
                        </p>
                    </div>
                </template>
            </div>

            <DesignInput
                v-model="playoutSearch"
                leading-icon="tabler:search"
                :placeholder="t('editorial.searchPlayoutEvents')"
            />

            <div v-if="loadingPlayoutEvents" class="flex flex-col gap-2">
                <DesignSkeleton
                    v-for="i in 4"
                    :key="i"
                    class="h-14 rounded-xl"
                />
            </div>

            <p
                v-else-if="filteredPlayoutEvents.length === 0"
                class="text-body-3 text-text-hint py-8 text-center"
            >
                {{ t("editorial.noPlayoutEvents") }}
            </p>

            <ul v-else class="flex max-h-96 flex-col gap-2 overflow-y-auto">
                <li v-for="e in filteredPlayoutEvents" :key="e.id">
                    <button
                        type="button"
                        class="bg-surface-raise gradient-border shadow-resting ds-focus-ring hover:bg-surface-indent flex w-full items-center justify-between gap-4 rounded-xl px-4 py-2.5 text-left transition-colors"
                        @click="selectPlayoutEvent(e.id)"
                    >
                        <div class="min-w-0">
                            <p class="text-title-2 text-text-default truncate">
                                {{ e.name || e.id }}
                            </p>
                            <p class="text-caption-1 text-text-hint truncate">
                                {{
                                    [e.date, e.status, e.productionUnit]
                                        .filter(Boolean)
                                        .join(" · ")
                                }}
                            </p>
                        </div>
                    </button>
                </li>
            </ul>
        </div>
    </DesignDialog>
</template>
