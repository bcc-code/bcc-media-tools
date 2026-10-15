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

const { t, te, locale } = useI18n();
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
    // what start/end showed when loaded, and their exact values: an untouched
    // field saves its original milliseconds instead of a rounded re-parse
    shown: { start: string; end: string; startMs: number; endMs: number };
    publishBmm: boolean;
    publishBcc: boolean;
    source: string;
    playoutKey: string;
    playoutSynced: boolean;
}

const session = ref<EditorialSession>();
const title = ref("");
const rows = ref<Row[]>([]);
const loading = ref(true);
const notFound = ref(false);

// A live session is logged in real time: no recording yet, times are this
// computer's clock (shown in Oslo time), and Playout fills in the details.
const isLive = computed(() => session.value?.kind === "live");
const clockStartMs = computed(() =>
    session.value?.clockStart
        ? timestampDate(session.value.clockStart).getTime()
        : 0,
);

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

const osloClock = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Europe/Oslo",
    hourCycle: "h23",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
});

// Live rows show the time of day; recording rows show the offset.
function displayMs(ms: number): string {
    if (!isLive.value) return formatMs(ms);
    return osloClock.format(new Date(clockStartMs.value + ms));
}

// Inverse of displayMs. A live time of day is read as the moment on or after
// the session's start, so a meeting can run past midnight.
function parseDisplay(text: string): number {
    if (!isLive.value) return parseTc(text);
    const base = parseTc(osloClock.format(new Date(clockStartMs.value)));
    let offset = parseTc(text) - base;
    // the clock start itself has sub-second precision the display drops
    offset -= clockStartMs.value % 1000;
    if (offset < -12 * 3600_000) offset += 24 * 3600_000;
    return offset;
}

function startMsOf(row: Row): number {
    return row.start === row.shown.start
        ? row.shown.startMs
        : parseDisplay(row.start);
}
function endMsOf(row: Row): number {
    return row.end === row.shown.end ? row.shown.endMs : parseDisplay(row.end);
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
    const startMs = Number(m.startMs);
    const endMs = Number(m.endMs);
    const start = displayMs(startMs);
    const end = displayMs(endMs);
    return {
        key: nextRowKey(),
        id: m.id,
        name: m.name,
        contributors: m.contributors,
        bibleVerses: m.bibleVerses,
        comment: m.comment,
        type: m.type,
        start,
        end,
        shown: { start, end, startMs, endMs },
        publishBmm: m.publishBmm,
        publishBcc: m.publishBcc,
        source: m.source || "manual",
        playoutKey: m.playoutKey,
        playoutSynced: m.playoutSynced,
    };
}

function durationOf(row: Row, i: number): string {
    if (isLive.value) {
        // A live row runs until the next one; the last is still running.
        const next = rows.value[i + 1];
        if (!next) return t("editorial.running");
        return formatMs(startMsOf(next) - startMsOf(row));
    }
    return formatMs(endMsOf(row) - startMsOf(row));
}

// Where a live row's details came from, for the "From" column.
function sourceLabel(row: Row): string {
    if (row.source === "import") return t("editorial.fromPlayout");
    if (row.playoutKey) return t("editorial.fromYouFilled");
    if (!row.name && !row.contributors) return t("editorial.fromYouWaiting");
    return t("editorial.fromYou");
}

const previewUrl = ref<string>();
const videoEl = useTemplateRef<HTMLVideoElement>("videoEl");

function preview(row: Row) {
    const el = videoEl.value;
    if (!el) return;
    el.currentTime = startMsOf(row) / 1000;
    void el.play();
}

// Highlight the marker whose [start, end) range contains the playhead.
const currentMs = ref(0);
function onTimeUpdate(e: Event) {
    currentMs.value = (e.target as HTMLVideoElement).currentTime * 1000;
}
const activeIndex = computed(() =>
    rows.value.findIndex((r) => {
        const start = startMsOf(r);
        const end = endMsOf(r);
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
        const start = startMsOf(m);
        const end = endMsOf(m);
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
        if (s.kind === "live") lastSyncedAt.value = Date.now();
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
        start: displayMs(0),
        end: displayMs(0),
        shown: { start: displayMs(0), end: displayMs(0), startMs: 0, endMs: 0 },
        publishBmm: false,
        publishBcc: false,
        source: "manual",
        playoutKey: "",
        playoutSynced: false,
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
const syncing = ref(false);
const attaching = ref(false);
const mutationInProgress = computed(
    () =>
        importing.value ||
        importingPlayout.value ||
        saving.value ||
        syncing.value ||
        attaching.value,
);

// Replace local state with what the server saved, without counting it as an
// unsaved edit.
async function applyServerSession(s: EditorialSession) {
    hydrated = false;
    session.value = s;
    title.value = s.title;
    rows.value = s.markers.map(toRow);
    if (s.previewUrl) previewUrl.value = s.previewUrl;
    await nextTick();
    dirty.value = false;
    hydrated = true;
}

// ── Live logging ──────────────────────────────────────────
const marking = ref(false);

// Saved at once with this computer's clock: the moment the editor saw.
async function markNow() {
    if (!isLive.value || !canEdit.value || marking.value) return;
    const at = new Date();
    marking.value = true;
    try {
        const m = await tracked(() =>
            api.addEditorialMarker({
                sessionId: sessionId.value,
                at: timestampFromDate(at),
            }),
        );
        // Adding a saved row is not an unsaved edit: keep sync running.
        const wasDirty = dirty.value;
        rows.value.push(toRow(m));
        await nextTick();
        dirty.value = wasDirty;
    } catch {
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    } finally {
        marking.value = false;
    }
}

function onKeydown(e: KeyboardEvent) {
    if (e.key !== "m" && e.key !== "M") return;
    if (e.metaKey || e.ctrlKey || e.altKey || e.repeat) return;
    const el = e.target as HTMLElement | null;
    if (
        el &&
        (el.isContentEditable ||
            ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName))
    )
        return;
    e.preventDefault();
    void markNow();
}

// Playout is re-read every two minutes while the session is open. Paused
// while there are unsaved edits (a Save would overwrite what arrived) or the
// editor is typing in a row.
const SYNC_INTERVAL_MS = 2 * 60_000;
const lastSyncedAt = ref<number>();
const syncFailed = ref(false);
const nowTick = ref(Date.now());
let syncTimer: ReturnType<typeof setInterval> | undefined;
let clockTimer: ReturnType<typeof setInterval> | undefined;

const nextSyncIn = computed(() => {
    if (!lastSyncedAt.value) return 0;
    return Math.max(0, lastSyncedAt.value + SYNC_INTERVAL_MS - nowTick.value);
});
const syncPaused = computed(
    () => dirty.value && effectiveMode.value === "edit",
);

function editingARow(): boolean {
    const el = document.activeElement as HTMLElement | null;
    return !!el && ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName);
}

async function syncNow() {
    if (!isLive.value || mutationInProgress.value) return;
    if (dirty.value) return;
    syncing.value = true;
    try {
        // Viewers can't write: they just pick up what editors synced.
        const s = canEdit.value
            ? await api.syncEditorialSessionFromPlayout({ id: sessionId.value })
            : await api.getEditorialSession({ id: sessionId.value });
        // Edits made while the request was out win; try again next round.
        if (!dirty.value) await applyServerSession(s);
        syncFailed.value = false;
    } catch {
        syncFailed.value = true;
    } finally {
        lastSyncedAt.value = Date.now();
        syncing.value = false;
    }
}

function scheduledSync() {
    if (editingARow()) return;
    void syncNow();
}

watch(isLive, (live) => {
    clearInterval(syncTimer);
    clearInterval(clockTimer);
    if (!live) return;
    syncTimer = setInterval(scheduledSync, SYNC_INTERVAL_MS);
    clockTimer = setInterval(() => (nowTick.value = Date.now()), 1000);
    void syncNow();
});
onMounted(() => window.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => {
    clearInterval(syncTimer);
    clearInterval(clockTimer);
    window.removeEventListener("keydown", onKeydown);
});

function formatCountdown(ms: number): string {
    const total = Math.ceil(ms / 1000);
    return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, "0")}`;
}

// ── Send to Mediabanken (attach the recording) ────────────
const attachOpen = ref(false);
const windowDialogOpen = computed({
    get: () => playoutPickerOpen.value || attachOpen.value,
    set: (open: boolean) => {
        if (open) return;
        playoutPickerOpen.value = false;
        attachOpen.value = false;
    },
});
const attachVxid = ref("");
// The VXID the shown window belongs to.
const attachLookedUp = ref("");

function openAttach() {
    if (mutationInProgress.value) return;
    attachVxid.value = "";
    attachLookedUp.value = "";
    recWindow.value = undefined;
    recAdjusting.value = false;
    attachOpen.value = true;
}

async function lookUpAttachWindow() {
    const vxid = attachVxid.value.trim();
    if (!vxid || vxid === attachLookedUp.value) return;
    attachLookedUp.value = vxid;
    await loadRecordingWindow(vxid);
}

// Rows whose clock time falls outside the recording: kept, but the editor
// should know before attaching.
const markersOutsideRecording = computed(() => {
    const w = recWindow.value;
    if (!w?.start || !w?.end || !isLive.value) return 0;
    const start = timestampDate(w.start).getTime();
    const end = timestampDate(w.end).getTime();
    return rows.value.filter((r) => {
        const at = clockStartMs.value + startMsOf(r);
        return at < start || at > end;
    }).length;
});

async function attachRecording() {
    const vxid = attachVxid.value.trim();
    if (!vxid || mutationInProgress.value) return;
    const range = adjustedWindow();
    if (!range) return;
    attaching.value = true;
    try {
        const s = await api.attachEditorialRecording({
            id: sessionId.value,
            VXID: vxid,
            recordingStart: range.start
                ? timestampFromDate(range.start)
                : undefined,
            recordingEnd: range.end ? timestampFromDate(range.end) : undefined,
        });
        attachOpen.value = false;
        await applyServerSession(s);
        toaster.create({ title: t("editorial.attached"), type: "success" });
    } catch (e) {
        toaster.create({
            title: t("editorial.attachFailed"),
            description: e instanceof Error ? e.message : undefined,
            type: "error",
        });
    } finally {
        attaching.value = false;
    }
}
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
    recDate.value = start ? osloDateInput(start) : "";
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

async function loadRecordingWindow(vxid = "") {
    loadingRecWindow.value = true;
    recAdjusting.value = false;
    recWindow.value = undefined;
    try {
        const w = await api.getRecordingWindow({
            sessionId: sessionId.value,
            VXID: vxid,
        });
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
    {
        value: "export",
        label: t("editorial.exportExcel"),
        icon: "tabler:file-spreadsheet",
        disabled: exporting.value,
    },
    ...(isLive.value && canEdit.value
        ? [
              {
                  value: "attach",
                  label: t("editorial.sendToMediabanken"),
                  icon: "tabler:upload",
                  disabled: mutationInProgress.value,
              },
          ]
        : []),
    ...(effectiveMode.value === "edit" && !isLive.value
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
    ...(canEdit.value
        ? [
              {
                  value: "delete",
                  label: t("editorial.delete"),
                  icon: "tabler:trash",
                  intent: "danger" as const,
              },
          ]
        : []),
]);
function onMenuSelect(value: string) {
    if (value === "delete") deleteOpen.value = true;
    else if (value === "import") void importMarkers();
    else if (value === "import-playout") void openPlayoutPicker();
    else if (value === "attach") openAttach();
    else if (value === "export") void exportExcel();
}

// ── Excel export ──────────────────────────────────────────
// The server builds the workbook from the saved session, so unsaved edits
// aren't in it; say so rather than export something that looks current.
const exporting = ref(false);

async function exportExcel() {
    if (exporting.value) return;
    exporting.value = true;
    try {
        const res = await api.exportEditorialSession({
            id: sessionId.value,
            language: locale.value,
        });
        const blob = new Blob([res.data as BlobPart], {
            type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = res.filename;
        a.click();
        URL.revokeObjectURL(url);
        if (dirty.value) {
            toaster.create({
                title: t("editorial.exportedSavedOnly"),
                type: "info",
            });
        }
    } catch {
        toaster.create({ title: t("editorial.exportFailed"), type: "error" });
    } finally {
        exporting.value = false;
    }
}

async function openPlayoutPicker() {
    if (mutationInProgress.value) return;
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

// The window typed into the adjust inputs, or an empty one when they weren't
// opened (the server then derives it). undefined: invalid, already reported.
function adjustedWindow(): { start?: Date; end?: Date } | undefined {
    if (!recAdjusting.value) return {};
    const invalid = () => {
        toaster.create({
            title: t("editorial.recordingWindowInvalid"),
            type: "error",
        });
        return undefined;
    };
    const start = osloToDate(recDate.value, recStartTime.value);
    if (!start) return invalid();
    let end: Date | undefined;
    if (recEndTime.value.trim()) {
        end = osloToDate(recDate.value, recEndTime.value);
        if (!end) return invalid();
        // An earlier end means the recording ran past midnight.
        if (end <= start) end = new Date(end.getTime() + 86400000);
    } else if (recNeedsEnd.value) {
        return invalid();
    }
    return { start, end };
}

function selectPlayoutEvent(eventId: string) {
    // Only send a window if the inputs were opened; otherwise the server
    // re-derives it, keeping the untouched case at two clicks.
    const range = adjustedWindow();
    if (!range) return;
    playoutPickerOpen.value = false;
    void importPlayoutMarkers(eventId, range.start, range.end);
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
                startMs: BigInt(startMsOf(r)),
                endMs: BigInt(endMsOf(r)),
                publishBmm: r.publishBmm,
                publishBcc: r.publishBcc,
                source: r.source,
                playoutKey: r.playoutKey,
            })),
        });
        await applyServerSession(s);
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
                <div v-else class="flex min-w-0 items-center gap-3">
                    <h1 class="text-heading-2 text-text-default truncate">
                        {{ title || session?.VXID }}
                    </h1>
                    <DesignBadge v-if="isLive" variant="warning">
                        {{ t("editorial.live") }}
                    </DesignBadge>
                </div>
                <p v-if="isLive" class="text-caption-1 text-text-hint mt-1">
                    {{ t("editorial.notInMediabanken") }}
                </p>
                <p
                    v-else-if="title && title !== session?.VXID"
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
                v-if="!loading && !notFound"
                class="mb-6 flex flex-wrap items-center gap-3"
            >
                <DesignSegmentGroup
                    v-if="canEdit"
                    v-model="modeModel"
                    :items="modeItems"
                    class="border-border-1 border"
                />
                <DesignButton
                    v-if="isLive && canEdit"
                    size="large"
                    icon="tabler:plus"
                    :loading="marking"
                    :title="t('editorial.markNowHint')"
                    @click="markNow"
                >
                    {{ t("editorial.markNow") }}
                    <span
                        class="text-caption-2 rounded-md border border-current px-1.5 opacity-70"
                        >M</span
                    >
                </DesignButton>
                <div class="ml-auto flex flex-wrap items-center gap-3">
                    <template v-if="isLive">
                        <span
                            class="text-caption-1 flex items-center gap-1.5"
                            :class="
                                syncFailed
                                    ? 'text-semantic-error'
                                    : 'text-text-hint'
                            "
                        >
                            <Icon
                                v-if="syncing"
                                name="svg-spinners:ring-resize"
                                class="size-4"
                            />
                            <Icon
                                v-else-if="syncFailed"
                                name="tabler:alert-triangle"
                                class="size-4"
                            />
                            <template v-if="syncing">
                                {{ t("editorial.syncing") }}
                            </template>
                            <template v-else-if="syncPaused">
                                {{ t("editorial.syncPaused") }}
                            </template>
                            <template v-else-if="syncFailed">
                                {{ t("editorial.syncFailed") }}
                            </template>
                            <template v-else-if="lastSyncedAt">
                                {{
                                    t("editorial.syncStatus", {
                                        time: osloTime.format(
                                            new Date(lastSyncedAt),
                                        ),
                                        next: formatCountdown(nextSyncIn),
                                    })
                                }}
                            </template>
                        </span>
                        <DesignButton
                            variant="tertiary"
                            icon="tabler:refresh"
                            class="border-border-1 border"
                            :disabled="mutationInProgress || dirty"
                            @click="syncNow"
                        >
                            {{ t("editorial.syncNow") }}
                        </DesignButton>
                    </template>
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
                    <DesignMenu
                        :items="menuItems"
                        :trigger-label="t('editorial.moreActions')"
                        @select="onMenuSelect"
                    />
                    <template v-if="canEdit">
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

            <div
                class="grid gap-6"
                :class="isLive ? '' : 'lg:grid-cols-[1fr_500px]'"
            >
                <div>
                    <p
                        v-if="rows.length === 0"
                        class="text-body-3 text-text-hint bg-surface-indent rounded-2xl px-4 py-10 text-center"
                    >
                        {{
                            isLive
                                ? t("editorial.noMarkersLive")
                                : t("editorial.noMarkers")
                        }}
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
                                        v-if="!isLive"
                                        class="border-border-1 w-10 border-b py-2 pl-2"
                                    ></th>
                                    <th
                                        v-else
                                        class="border-border-1 border-b py-2 pl-3 font-normal"
                                    >
                                        {{ t("editorial.col.time") }}
                                    </th>
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
                                        v-if="
                                            effectiveMode === 'edit' && !isLive
                                        "
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
                                        v-if="isLive"
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.from") }}
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
                                    <td
                                        v-if="isLive"
                                        class="py-2 pl-3 whitespace-nowrap"
                                    >
                                        <DesignInput
                                            v-if="effectiveMode === 'edit'"
                                            v-model="row.start"
                                            class="w-28"
                                        />
                                        <span
                                            v-else
                                            class="text-body-3 text-text-default tabular-nums"
                                        >
                                            {{ row.start }}
                                        </span>
                                    </td>
                                    <td v-else class="py-2 pl-2">
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
                                        {{ durationOf(row, i) }}
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
                                        v-if="
                                            effectiveMode === 'edit' && !isLive
                                        "
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
                                        v-if="isLive"
                                        class="text-caption-1 text-text-muted px-2 py-2 whitespace-nowrap"
                                    >
                                        {{ sourceLabel(row) }}
                                    </td>
                                    <td
                                        v-if="effectiveMode === 'edit'"
                                        class="py-2 pl-2"
                                    >
                                        <div class="flex items-center gap-0.5">
                                            <button
                                                v-if="!isLive"
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
                        v-if="effectiveMode === 'edit' && !isLive"
                        variant="tertiary"
                        icon="tabler:plus"
                        class="mt-3"
                        @click="addRow"
                    >
                        {{ t("editorial.addRow") }}
                    </DesignButton>
                </div>

                <div v-if="!isLive">
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
                                :min="startMsOf(activeMarker)"
                                :max="endMsOf(activeMarker)"
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

    <!--
        Shared by the Playout import and "Send to Mediabanken": both need the
        recording's window on the wall clock, and both let the editor fix it.
    -->
    <DesignDialog
        v-model:open="windowDialogOpen"
        :title="
            attachOpen
                ? t('editorial.sendToMediabanken')
                : t('editorial.selectPlayoutEvent')
        "
        size="lg"
    >
        <div class="flex flex-col gap-3">
            <template v-if="attachOpen">
                <p class="text-body-3 text-text-muted">
                    {{ t("editorial.attachDescription") }}
                </p>
                <form
                    class="flex items-end gap-2"
                    @submit.prevent="lookUpAttachWindow"
                >
                    <DesignInput
                        v-model="attachVxid"
                        :label="t('editorial.recordingVxid')"
                        placeholder="VX-123456"
                        class="grow"
                        @change="lookUpAttachWindow"
                    />
                    <DesignButton
                        type="submit"
                        variant="secondary"
                        :disabled="!attachVxid.trim()"
                    >
                        {{ t("editorial.lookUp") }}
                    </DesignButton>
                </form>
            </template>
            <!--
                The window the import will use. Above the list because picking
                an event imports at once — the only moment to catch a bad one.
            -->
            <div
                v-if="!attachOpen || attachLookedUp"
                class="flex flex-col gap-2"
            >
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
                                placeholder="HH:MM"
                                :invalid="
                                    !!recStartTime && !isClockTime(recStartTime)
                                "
                                :label="t('editorial.recordingStartTime')"
                                class="min-w-28 grow"
                            />
                            <DesignInput
                                v-model="recEndTime"
                                placeholder="HH:MM"
                                :invalid="
                                    !!recEndTime && !isClockTime(recEndTime)
                                "
                                :label="t('editorial.recordingEndTime')"
                                class="min-w-28 grow"
                            />
                        </div>
                        <p class="text-caption-1 text-text-hint">
                            {{
                                attachOpen
                                    ? t("editorial.attachWindowHint")
                                    : t("editorial.recordingWindowHint")
                            }}
                        </p>
                    </div>
                </template>
            </div>

            <template v-if="attachOpen">
                <DesignBanner
                    v-if="markersOutsideRecording"
                    variant="warning"
                    icon="tabler:alert-triangle"
                >
                    {{
                        t("editorial.markersOutsideRecording", {
                            n: markersOutsideRecording,
                        })
                    }}
                </DesignBanner>
                <div class="flex justify-end gap-2">
                    <DesignButton
                        variant="secondary"
                        @click="attachOpen = false"
                    >
                        {{ t("editorial.cancel") }}
                    </DesignButton>
                    <DesignButton
                        :loading="attaching"
                        :disabled="
                            !attachLookedUp ||
                            attachLookedUp !== attachVxid.trim() ||
                            loadingRecWindow ||
                            (!recWindow?.start && !recAdjusting)
                        "
                        @click="attachRecording"
                    >
                        {{ t("editorial.attachRecording") }}
                    </DesignButton>
                </div>
            </template>
            <EditorialPlayoutEventList
                v-else
                :events="playoutEvents"
                :loading="loadingPlayoutEvents"
                @select="selectPlayoutEvent($event.id)"
            />
        </div>
    </DesignDialog>
</template>
