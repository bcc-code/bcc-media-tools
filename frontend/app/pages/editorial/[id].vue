<script setup lang="ts">
import type {
    EditorialSession,
    EditorialMarker,
    EditorialPresence,
    PlayoutEvent,
    GetRecordingWindowResponse,
} from "~~/src/gen/api/v1/api_pb";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import type {
    MobileMarker,
    MarkerDetails,
} from "~/components/editorial/MobileMarkerList.vue";
import { timestampDate, timestampFromDate } from "@bufbuild/protobuf/wkt";

const route = useRoute("editorial-id");
const sessionId = computed(() => route.params.id as string);
const config = useRuntimeConfig();

// The session's preview_url is a path on our own server — the proxy that
// fetches the video from Cantemo, which wants an auth header the browser will
// not send and whose host only resolves on the internal network. Like the
// other tools' media URLs, it needs the API base in front of it.
function previewSrc(path: string | undefined): string | undefined {
    return path ? `${config.public.grpcUrl}${path}` : undefined;
}

const { t, te, locale } = useI18n();
const api = useAPI();
const perms = usePermissions();
const toaster = useToast();

// Live and up: the edit view, live logging, Playout, attaching the recording.
const canLive = perms.canRunLiveEditorial;
// Editor and up: contributors, type and Bible verses on existing markers.
const canEditDetails = perms.canEditEditorialDetails;

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
// Phones get the card layout, which is simple mode: the edit table doesn't fit.
const isPhone = useMediaQuery("(max-width: 767px)");
const effectiveMode = computed(() =>
    canLive.value && !isPhone.value ? mode.value : "simple",
);

// Whether there is a preview column beside the list: a live session has no
// asset to show yet, and phones get the card layout instead.
const hasPreview = computed(() => !isLive.value && !isPhone.value);

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
        previewUrl.value = previewSrc(s.previewUrl);
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
        // Simple-mode changes are saved as they're made, so only edit mode
        // has unsaved changes.
        if (hydrated && effectiveMode.value === "edit") dirty.value = true;
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
// ── Phone layout ──────────────────────────────────────────
const mobileMarkers = computed<MobileMarker[]>(() => {
    const items = rows.value.map((r, i) => ({
        key: r.key,
        time: r.start,
        duration: durationOf(r, i),
        type: r.type,
        typeLabel: r.type ? typeLabel(r.type) : "",
        name: r.name,
        contributors: r.contributors,
        bibleVerses: r.bibleVerses,
        comment: r.comment,
        publishBmm: r.publishBmm,
        publishBcc: r.publishBcc,
        commentingBy: r.id ? commentingOn(r.id) : "",
    }));
    // Live: the marker just logged is the one to comment on.
    return isLive.value ? items.reverse() : items;
});

function rowByKey(key: string): Row | undefined {
    return rows.value.find((r) => r.key === key);
}

async function onMobileComment(key: string, comment: string) {
    const row = rowByKey(key);
    if (!row?.id) return;
    const before = row.comment;
    row.comment = comment;
    try {
        await tracked(() =>
            api.setEditorialComment({
                sessionId: sessionId.value,
                markerId: row.id,
                comment,
            }),
        );
    } catch {
        row.comment = before;
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    }
}

// ── Details (contributors, type, Bible verses; editor and up) ──
// Simple and phone views save each change on its own; edit mode saves on Save.
type Details = Pick<
    Row,
    "name" | "contributors" | "type" | "bibleVerses" | "playoutSynced"
>;
function snapshot(row: Row): Details {
    const { name, contributors, type, bibleVerses, playoutSynced } = row;
    return { name, contributors, type, bibleVerses, playoutSynced };
}

// Only the phone drawer edits details now — the table shows them read-only in
// simple mode and saves on the batch Save in edit mode — so the caller always
// has the pre-edit snapshot on hand.
async function persistDetails(row: Row, before: Details) {
    if (effectiveMode.value === "edit" || !row.id) return;
    const changed =
        row.name !== before.name ||
        row.contributors !== before.contributors ||
        row.type !== before.type ||
        row.bibleVerses !== before.bibleVerses;
    if (!changed) return;
    // A hand edit stops Playout from refreshing the row.
    row.playoutSynced = false;
    try {
        await tracked(() =>
            api.setEditorialDetails({
                sessionId: sessionId.value,
                markerId: row.id,
                name: row.name,
                contributors: row.contributors,
                type: row.type,
                bibleVerses: row.bibleVerses,
            }),
        );
    } catch {
        Object.assign(row, before);
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    }
}

function onMobileTitle(key: string, name: string) {
    const row = rowByKey(key);
    if (!row) return;
    nameBeforeEdit.value = row.name;
    row.name = name;
    void persistName(row);
}

function onMobileDetails(key: string, details: MarkerDetails) {
    const row = rowByKey(key);
    if (!row) return;
    const before = snapshot(row);
    Object.assign(row, details);
    void persistDetails(row, before);
}

async function takeSync() {
    try {
        await api.takeEditorialSync({ sessionId: sessionId.value, clientId });
        syncClientId.value = clientId;
        void syncNow();
    } catch {
        toaster.create({ title: t("editorial.syncFailed"), type: "error" });
    }
}

// key: the marker whose comment field has focus; null when none.
function onMobileCommenting(key: string | null) {
    if (key) onCommentFocus(rowByKey(key));
    else onCommentBlur();
}

// Throw away this tab's unsaved edit-view changes and take the session as
// it is now.
async function discardAndReload() {
    remoteChanged.value = false;
    dirty.value = false;
    await applyServerSession(
        await api.getEditorialSession({ id: sessionId.value }),
    );
}

function onMobilePublish(key: string, target: "bmm" | "bcc", value: boolean) {
    const row = rowByKey(key);
    if (row) void onPublishToggle(row, target, value);
}

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
    if (s.previewUrl) previewUrl.value = previewSrc(s.previewUrl);
    await nextTick();
    dirty.value = false;
    hydrated = true;
}

// ── Multiplayer ───────────────────────────────────────────
// Every open page follows the session over a server stream, so someone
// marking and someone commenting see each other's changes as they happen.
// Changes arrive as the whole session and are merged row by row, leaving the
// field you are typing in alone.
const clientId =
    globalThis.crypto?.randomUUID?.() ??
    `tab-${Math.random().toString(36).slice(2)}`;
const presence = ref<EditorialPresence[]>([]);
// The tab that runs the Playout sync for everyone ("" when no live tab is
// connected). Only it syncs; other live users can take over.
const syncClientId = ref("");
const liveConnected = ref(false);
// Others changed the session while this tab has unsaved edit-view changes.
const remoteChanged = ref(false);
let unmounted = false;
let watchAbort: AbortController | undefined;
let fallbackTimer: ReturnType<typeof setInterval> | undefined;

// Merge a session from the server into the page without counting it as an
// edit. Rows keep their keys (and so their DOM and focus); the row being
// typed in keeps its text fields until the field is left.
async function mergeServerSession(s: EditorialSession) {
    if (effectiveMode.value === "edit" && dirty.value) {
        remoteChanged.value = true;
        return;
    }
    const focusedKey = (document.activeElement as HTMLElement | null)
        ?.closest("[data-row-key]")
        ?.getAttribute("data-row-key");
    const byId = new Map(rows.value.filter((r) => r.id).map((r) => [r.id, r]));

    hydrated = false;
    session.value = s;
    if (!document.activeElement?.closest("[data-session-title]"))
        title.value = s.title;
    if (s.previewUrl) previewUrl.value = previewSrc(s.previewUrl);
    rows.value = s.markers.map((m) => {
        const fresh = toRow(m);
        const row = byId.get(m.id);
        if (!row) return fresh;
        if (row.key === focusedKey) {
            const { name, contributors, bibleVerses, comment, type } = row;
            Object.assign(row, fresh, {
                key: row.key,
                name,
                contributors,
                bibleVerses,
                comment,
                type,
            });
        } else {
            Object.assign(row, fresh, { key: row.key });
        }
        return row;
    });
    await nextTick();
    hydrated = true;
}

async function watchSession() {
    let backoff = 1000;
    while (!unmounted) {
        watchAbort = new AbortController();
        try {
            for await (const ev of api.watchEditorialSession(
                { id: sessionId.value, clientId },
                { signal: watchAbort.signal },
            )) {
                liveConnected.value = true;
                backoff = 1000;
                if (ev.deleted) {
                    toaster.create({
                        title: t("editorial.deletedByOther"),
                        type: "info",
                    });
                    dirty.value = false;
                    await navigateTo("/editorial/");
                    return;
                }
                // Keep-alives carry neither; presence always includes us.
                if (ev.session || ev.presence.length) {
                    presence.value = ev.presence;
                    syncClientId.value = ev.syncClientId;
                }
                if (ev.session) await mergeServerSession(ev.session);
            }
        } catch {
            if (unmounted) return;
        }
        liveConnected.value = false;
        await new Promise((r) => setTimeout(r, backoff));
        backoff = Math.min(backoff * 2, 30_000);
    }
}

// Without the stream (a proxy that doesn't allow it, a dropped connection),
// fall back to reading the session every few seconds.
async function pollFallback() {
    if (liveConnected.value || loading.value || notFound.value) return;
    try {
        await mergeServerSession(
            await api.getEditorialSession({ id: sessionId.value }),
        );
    } catch {
        // The next round tries again.
    }
}

onMounted(() => {
    void watchSession();
    fallbackTimer = setInterval(pollFallback, 5000);
});
onBeforeUnmount(() => {
    unmounted = true;
    watchAbort?.abort();
    clearInterval(fallbackTimer);
});

// What this tab is doing, for the others. Only sent when it changes.
let sentPresence = "viewing|";
function reportPresence(
    activity: "viewing" | "commenting" | "editing",
    markerId = "",
) {
    const key = `${activity}|${markerId}`;
    if (key === sentPresence) return;
    sentPresence = key;
    api.setEditorialPresence({
        sessionId: sessionId.value,
        clientId,
        activity,
        markerId,
    }).catch(() => {
        sentPresence = "";
    });
}

function onCommentFocus(row: Row | undefined) {
    if (row?.id) reportPresence("commenting", row.id);
}
function onCommentBlur() {
    reportPresence(
        effectiveMode.value === "edit" && dirty.value ? "editing" : "viewing",
    );
}
watch(
    () => effectiveMode.value === "edit" && dirty.value,
    (editing) => reportPresence(editing ? "editing" : "viewing"),
);

// "kari.nordmann@bcc.media" → "Kari Nordmann"
function displayName(email: string): string {
    return email
        .split("@")[0]!
        .split(/[._-]+/)
        .filter(Boolean)
        .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
        .join(" ");
}
function initials(email: string): string {
    return displayName(email)
        .split(" ")
        .map((p) => p.charAt(0))
        .slice(0, 2)
        .join("");
}

// The others in the session, one entry per person.
const others = computed(() => {
    const seen = new Map<string, EditorialPresence>();
    for (const p of presence.value) {
        if (p.clientId === clientId) continue;
        const prev = seen.get(p.email);
        if (!prev || prev.activity === "viewing") seen.set(p.email, p);
    }
    return [...seen.values()];
});
const othersEditing = computed(() =>
    others.value.filter((p) => p.activity === "editing"),
);

// Who is commenting on a marker, by marker id ("Kari", "Kari and Ola").
function commentingOn(markerId: string): string {
    const names = presence.value
        .filter(
            (p) =>
                p.clientId !== clientId &&
                p.activity === "commenting" &&
                p.markerId === markerId,
        )
        .map((p) => displayName(p.email).split(" ")[0]!);
    return [...new Set(names)].join(", ");
}

// ── Live logging ──────────────────────────────────────────
const marking = ref(false);

// Saved at once with this computer's clock: the moment the editor saw.
async function markNow() {
    if (!isLive.value || !canLive.value || marking.value) return;
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
        // The stream may have delivered it already.
        if (!rows.value.some((r) => r.id === m.id)) rows.value.push(toRow(m));
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

// Without a connection nobody is known to hold the lock, so try: the server
// refuses if another tab holds it.
const holdsSync = computed(
    () =>
        canLive.value &&
        (syncClientId.value === clientId || syncClientId.value === ""),
);
// Where the sync runs, for the other live users.
const syncHolderLabel = computed(() => {
    const holder = presence.value.find(
        (p) => p.clientId === syncClientId.value,
    );
    if (!holder) return "";
    const me = presence.value.find((p) => p.clientId === clientId);
    return me && holder.email === me.email
        ? t("editorial.syncedInYourOtherTab")
        : t("editorial.syncedBy", { name: displayName(holder.email) });
});

async function syncNow() {
    if (!isLive.value || !holdsSync.value || mutationInProgress.value) return;
    if (dirty.value) return;
    syncing.value = true;
    try {
        // Everyone else gets the result through the stream.
        const s = await api.syncEditorialSessionFromPlayout({
            id: sessionId.value,
            clientId,
        });
        await mergeServerSession(s);
        syncFailed.value = false;
    } catch (e) {
        // Another tab holds the lock: not a failure, it syncs for us.
        syncFailed.value =
            ConnectError.from(e).code !== Code.FailedPrecondition;
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
    ...(isLive.value && canLive.value
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
    ...(perms.isEditorialAdmin.value
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
    if (canLive.value && dirty.value) {
        return window.confirm(t("editorial.unsavedWarning"));
    }
});
</script>

<template>
    <div v-if="!perms.canUseEditorial.value" class="py-16 text-center">
        <p class="text-body-2 text-text-muted">{{ t("noPermissions") }}</p>
    </div>

    <div
        v-else
        class="mx-auto w-full max-w-[1700px] px-4 py-6"
        :class="effectiveMode === 'simple' ? 'large-type' : ''"
    >
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
                    data-session-title
                    :placeholder="session?.VXID"
                />
                <div v-else class="flex min-w-0 items-center gap-3">
                    <h1 class="text-heading-2 text-text-default truncate">
                        {{ title || session?.VXID || t("editorial.untitled") }}
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
                    v-if="canLive && !isPhone"
                    v-model="modeModel"
                    :items="modeItems"
                    class="border-border-1 border"
                />
                <DesignButton
                    v-if="isLive && canLive"
                    size="large"
                    icon="tabler:plus"
                    class="grow md:grow-0"
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
                <!--
                    Next to Mark now on a phone, where the status row below
                    wraps; last in the toolbar on wider screens.
                -->
                <div class="flex max-md:ml-auto md:order-last">
                    <DesignMenu
                        :items="menuItems"
                        :trigger-label="t('editorial.moreActions')"
                        @select="onMenuSelect"
                    />
                </div>
                <div
                    class="flex flex-wrap items-center gap-3 max-md:w-full md:ml-auto"
                >
                    <div
                        v-if="others.length"
                        class="flex -space-x-1.5"
                        :aria-label="
                            t('editorial.othersHere', {
                                names: others
                                    .map((p) => displayName(p.email))
                                    .join(', '),
                            })
                        "
                    >
                        <span
                            v-for="p in others"
                            :key="p.clientId"
                            :title="displayName(p.email)"
                            class="bg-primary-default text-on-primary text-caption-2 ring-surface-default flex size-7 items-center justify-center rounded-full ring-2"
                        >
                            {{ initials(p.email) }}
                        </span>
                    </div>
                    <span
                        class="text-caption-1 flex items-center gap-1.5"
                        :class="
                            liveConnected
                                ? 'text-text-hint'
                                : 'text-semantic-warning'
                        "
                        :title="
                            liveConnected
                                ? t('editorial.connectedHint')
                                : t('editorial.reconnectingHint')
                        "
                    >
                        <span
                            class="size-2 rounded-full"
                            :class="
                                liveConnected
                                    ? 'bg-semantic-success'
                                    : 'bg-semantic-warning'
                            "
                        />
                        {{
                            liveConnected
                                ? t("editorial.connected")
                                : t("editorial.reconnecting")
                        }}
                    </span>
                    <template v-if="isLive && canLive && !holdsSync">
                        <span class="text-caption-1 text-text-hint">
                            {{ syncHolderLabel }}
                        </span>
                        <DesignButton
                            variant="tertiary"
                            size="small"
                            class="border-border-1 border"
                            @click="takeSync"
                        >
                            {{ t("editorial.takeOverSync") }}
                        </DesignButton>
                    </template>
                    <template v-if="isLive && holdsSync">
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
                    <template v-if="canLive">
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
                v-if="othersEditing.length || remoteChanged"
                class="mb-4 flex flex-col gap-2"
            >
                <DesignBanner
                    v-if="othersEditing.length"
                    variant="warning"
                    icon="tabler:users"
                >
                    {{
                        t("editorial.otherIsEditing", {
                            name: othersEditing
                                .map((p) => displayName(p.email))
                                .join(", "),
                        })
                    }}
                </DesignBanner>
                <DesignBanner
                    v-if="remoteChanged"
                    variant="warning"
                    icon="tabler:alert-triangle"
                >
                    <div class="flex flex-wrap items-center gap-3">
                        <span class="grow">
                            {{ t("editorial.remoteChanged") }}
                        </span>
                        <DesignButton
                            variant="secondary"
                            size="small"
                            @click="discardAndReload"
                        >
                            {{ t("editorial.discardAndReload") }}
                        </DesignButton>
                    </div>
                </DesignBanner>
            </div>

            <div
                class="grid gap-6"
                :class="isLive ? '' : 'lg:grid-cols-[1fr_500px]'"
            >
                <div class="min-w-0">
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

                    <template v-if="rows.length && isPhone">
                        <p class="text-caption-1 text-text-hint mb-2">
                            {{
                                t("editorial.commentSummary", {
                                    n: rows.length,
                                    commented: rows.filter((r) => r.comment)
                                        .length,
                                })
                            }}
                        </p>
                        <EditorialMobileMarkerList
                            :markers="mobileMarkers"
                            :can-edit="canEditDetails"
                            :type-items="typeItems"
                            @comment="onMobileComment"
                            @details="onMobileDetails"
                            @commenting="onMobileCommenting"
                            @title="onMobileTitle"
                            @publish="onMobilePublish"
                        />
                    </template>
                    <!--
                        Horizontal scroll for the edit table's dozen columns
                        only. It has to come off in simple view, because
                        overflow-x makes this a scroll container on BOTH axes
                        (the spec turns the other axis's `visible` into `auto`),
                        and the sticky headers below would then stick to a box
                        that never scrolls instead of to the page. Simple view's
                        five columns fit the width anyway.
                    -->
                    <div
                        v-else-if="rows.length"
                        :class="
                            effectiveMode === 'edit' ? 'overflow-x-auto' : ''
                        "
                    >
                        <table class="w-full border-separate border-spacing-0">
                            <!--
                                Parks under the app header, which is itself
                                sticky at the top; below its z-10, so the header
                                wins wherever they meet. The fill is the body's
                                own background (nuxt.config bodyAttrs) rather
                                than a surface token, so the row reads exactly
                                as it did unstuck — it is only there to stop the
                                rows showing through as they pass under.
                            -->
                            <thead
                                class="text-caption-1 text-text-hint text-left"
                                :class="
                                    effectiveMode === 'simple'
                                        ? '[&>tr>th]:sticky [&>tr>th]:top-[var(--header-height)] [&>tr>th]:z-[5] [&>tr>th]:bg-neutral-100 dark:[&>tr>th]:bg-neutral-950'
                                        : ''
                                "
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
                                        :class="
                                            effectiveMode === 'simple'
                                                ? 'w-[42%]'
                                                : ''
                                        "
                                    >
                                        {{ t("editorial.col.title") }}
                                    </th>
                                    <th
                                        v-if="effectiveMode === 'edit'"
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.contributors") }}
                                    </th>
                                    <th
                                        v-if="effectiveMode === 'edit'"
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.bibleVerses") }}
                                    </th>
                                    <th
                                        v-if="effectiveMode === 'edit'"
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.duration") }}
                                    </th>
                                    <th
                                        v-if="effectiveMode === 'edit'"
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
                                    :data-row-key="row.key"
                                    class="[&>td]:border-border-1/50 transition-colors [&>td]:border-b"
                                    :class="[
                                        i === activeIndex
                                            ? 'bg-primary-default/10'
                                            : '',
                                        i === dragIndex ? 'opacity-40' : '',
                                        dragOverIndex === i && dragIndex !== i
                                            ? '[&>td]:border-t-primary-default [&>td]:border-t'
                                            : '',
                                        // The title cell is the tall one once
                                        // Medvirkende rides under it, and
                                        // centring the rest against the whole
                                        // stack drops them below the title.
                                        // Topped out they line up with its
                                        // first line: the cells share py-2, so
                                        // the switches and the play button land
                                        // within a pixel or two of it.
                                        effectiveMode === 'simple'
                                            ? '[&>td]:align-top'
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
                                            v-if="effectiveMode === 'edit'"
                                            v-model="row.name"
                                            @focusin="nameBeforeEdit = row.name"
                                            @change="persistName(row)"
                                        />
                                        <!--
                                            Simple view wraps the title instead
                                            of truncating it, and stands the
                                            contributor in as the placeholder:
                                            most rows are a contribution with no
                                            title at all, and the name belongs on
                                            the line reviewers scan rather than
                                            in a footnote under an empty box.
                                            Typing a title replaces it.
                                        -->
                                        <EditorialGrowingTextarea
                                            v-else
                                            v-model="row.name"
                                            :placeholder="row.contributors"
                                            placeholder-is-value
                                            :aria-label="
                                                t('editorial.col.title')
                                            "
                                            class="title-field"
                                            @focusin="nameBeforeEdit = row.name"
                                            @change="persistName(row)"
                                        />
                                        <!--
                                            Medvirkende and Bibelvers ride under
                                            the title rather than taking columns
                                            of their own: both are empty on most
                                            rows, and a column that is blank
                                            most of the way down just holds the
                                            row apart. Read-only here; both are
                                            set in Edit mode.

                                            A line each, rather than joined: a
                                            talk can carry ten references, and
                                            run together the name was just the
                                            start of a long muted run. The name
                                            also takes the heavier weight — it
                                            is who the row is, and the reviewer
                                            scans for it.

                                            The name is skipped where the title
                                            field is already standing it in as
                                            its placeholder.
                                        -->
                                        <p
                                            v-if="
                                                effectiveMode === 'simple' &&
                                                row.name &&
                                                row.contributors
                                            "
                                            class="text-body-3 text-text-muted mt-1 px-3 font-medium"
                                        >
                                            {{ row.contributors }}
                                        </p>
                                        <p
                                            v-if="
                                                effectiveMode === 'simple' &&
                                                row.bibleVerses
                                            "
                                            class="text-body-3 text-text-muted mt-0.5 px-3 tabular-nums"
                                        >
                                            {{ row.bibleVerses }}
                                        </p>
                                    </td>
                                    <td
                                        v-if="effectiveMode === 'edit'"
                                        class="px-2 py-2"
                                    >
                                        <DesignInput
                                            v-model="row.contributors"
                                        />
                                    </td>
                                    <td
                                        v-if="effectiveMode === 'edit'"
                                        class="px-2 py-2"
                                    >
                                        <DesignInput
                                            v-model="row.bibleVerses"
                                        />
                                    </td>
                                    <td
                                        v-if="effectiveMode === 'edit'"
                                        class="text-body-3 text-text-muted px-2 py-2 tabular-nums"
                                    >
                                        {{ durationOf(row, i) }}
                                    </td>
                                    <td
                                        v-if="effectiveMode === 'edit'"
                                        class="px-2 py-2"
                                    >
                                        <DesignSelect
                                            v-model="row.type"
                                            :items="typeItems"
                                        />
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
                                            commentBeforeEdit = row.comment;
                                            onCommentFocus(row);
                                        "
                                        @focusout="onCommentBlur"
                                        @change="persistComment(row)"
                                    >
                                        <DesignInput
                                            v-if="effectiveMode === 'edit'"
                                            v-model="row.comment"
                                        />
                                        <!--
                                            The comment is the one thing every
                                            reviewer writes, and the notes run
                                            longer than the column — so in
                                            simple view it wraps rather than
                                            scrolling out of sight. No
                                            placeholder: the column header
                                            already says Kommentar, and
                                            repeating it down every row of the
                                            table only adds noise. The phone
                                            cards keep theirs — a card has no
                                            header to label the field.
                                        -->
                                        <EditorialGrowingTextarea
                                            v-else
                                            v-model="row.comment"
                                            :aria-label="
                                                t('editorial.col.comment')
                                            "
                                        />
                                        <p
                                            v-if="
                                                row.id && commentingOn(row.id)
                                            "
                                            class="text-caption-2 text-text-hint mt-1 flex items-center gap-1"
                                        >
                                            <Icon
                                                name="tabler:pencil"
                                                class="size-3"
                                            />
                                            {{
                                                t("editorial.isCommenting", {
                                                    name: commentingOn(row.id),
                                                })
                                            }}
                                        </p>
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
                <div v-if="hasPreview">
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

<!--
    The simple view is the one the editorial reviewers sit with for hours, and
    several of them read it with tired eyes. Overriding the design system's
    type-scale tokens on this subtree lifts every size at once — including the
    text inside Design* components, which hard-code the token classes — instead
    of rewriting a class on each of the hundred-odd elements below. Edit mode
    keeps the Figma scale: it is a power-user view and needs the density.
-->
<style scoped>
.large-type {
    --text-heading-2: 36px;
    --text-heading-3: 25px;

    --text-title-1: 22px;
    --text-title-2: 20px;
    --text-title-3: 18px;

    --text-body-1: 21px;
    --text-body-2: 19px;
    --text-body-2--line-height: 1.35;
    --text-body-3: 17px;
    --text-body-3--line-height: 1.35;

    --text-caption-1: 14px;
    --text-caption-1--line-height: 1.4;
    --text-caption-2: 13px;
    --text-caption-2--line-height: 1.4;
}

/*
    The title carries the row, so it steps up once more above the body text
    around it. DesignInput hard-codes text-body-3 on its own input, so the
    token is the way in.
*/
.title-field {
    --text-body-3: 20px;
}
</style>
