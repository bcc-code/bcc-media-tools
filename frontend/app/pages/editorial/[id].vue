<script setup lang="ts">
import type {
    EditorialSession,
    EditorialMarker,
} from "~~/src/gen/api/v1/api_pb";

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

// Translated label for a stored type value. Falls back to the raw value
// (capitalized) so legacy types no longer in TYPE_OPTIONS still render.
function typeLabel(type: string): string {
    const key = `editorial.types.${type}`;
    if (te(key)) return t(key);
    return type ? type.charAt(0).toUpperCase() + type.slice(1) : type;
}

// Select options with translated labels; values stay the stored lowercase keys.
const typeItems = computed(() =>
    TYPE_OPTIONS.map((value) => ({ label: typeLabel(value), value })),
);

// A single editable row. Start/End are kept as "HH:MM:SS" strings so text
// editing is natural; they're parsed to milliseconds only at save/preview.
interface Row {
    id: string;
    name: string;
    contributors: string;
    comment: string;
    type: string;
    start: string;
    end: string;
    publish: boolean;
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

function toRow(m: EditorialMarker): Row {
    return {
        id: m.id,
        name: m.name,
        contributors: m.contributors,
        comment: m.comment,
        type: m.type,
        start: formatMs(Number(m.startMs)),
        end: formatMs(Number(m.endMs)),
        publish: m.publish,
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

// ── Publish toggle ────────────────────────────────────────
async function onPublishToggle(row: Row, value: boolean) {
    row.publish = value;
    // In edit mode the change is persisted on Save. In simple mode there is no
    // Save button, so persist the single toggle immediately.
    if (effectiveMode.value === "edit") return;
    try {
        await api.setEditorialPublish({
            sessionId: sessionId.value,
            markerId: row.id,
            publish: value,
        });
    } catch {
        row.publish = !value;
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    }
}

// ── Edit-mode mutations ───────────────────────────────────
function addRow() {
    rows.value.push({
        id: "",
        name: "",
        contributors: "",
        comment: "",
        type: "",
        start: "00:00:00",
        end: "00:00:00",
        publish: false,
        source: "manual",
    });
}

function removeRow(i: number) {
    rows.value.splice(i, 1);
}

function move(i: number, delta: number) {
    const j = i + delta;
    if (j < 0 || j >= rows.value.length) return;
    const [item] = rows.value.splice(i, 1);
    rows.value.splice(j, 0, item!);
}

// ── Backend actions ───────────────────────────────────────
const importing = ref(false);
const saving = ref(false);
const deleteOpen = ref(false);

// Secondary/destructive actions live in the overflow menu; Save stays primary.
const menuItems = computed(() => [
    ...(effectiveMode.value === "edit"
        ? [
              {
                  value: "import",
                  label: t("editorial.import"),
                  icon: "tabler:download",
                  disabled: importing.value,
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
}

async function importMarkers() {
    importing.value = true;
    try {
        const res = await api.importEditorialMarkers({ id: sessionId.value });
        for (const m of res.markers) rows.value.push(toRow(m));
        dirty.value = true;
        toaster.create({
            title: t("editorial.importedCount", { n: res.markers.length }),
            type: "success",
        });
    } catch {
        toaster.create({ title: t("editorial.importFailed"), type: "error" });
    } finally {
        importing.value = false;
    }
}

async function save() {
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
                comment: r.comment,
                type: r.type,
                startMs: BigInt(parseTc(r.start)),
                endMs: BigInt(parseTc(r.end)),
                publish: r.publish,
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
            </div>

            <div
                v-if="canEdit"
                class="mb-6 flex items-center justify-between gap-3"
            >
                <DesignSegmentGroup
                    v-model="modeModel"
                    :items="modeItems"
                    class="border-border-1 border"
                />
                <div class="flex items-center gap-2">
                    <DesignMenu
                        :items="menuItems"
                        :trigger-label="t('editorial.moreActions')"
                        @select="onMenuSelect"
                    />
                    <DesignButton
                        v-if="effectiveMode === 'edit'"
                        icon="tabler:device-floppy"
                        :loading="saving"
                        @click="save"
                    >
                        {{ t("editorial.save") }}
                    </DesignButton>
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
                    <div v-else class="overflow-x-auto">
                        <table class="w-full border-separate border-spacing-0">
                            <thead
                                class="text-caption-1 text-text-hint text-left"
                            >
                                <tr>
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
                                        {{ t("editorial.col.duration") }}
                                    </th>
                                    <th
                                        class="border-border-1 border-b px-2 py-2 font-normal"
                                    >
                                        {{ t("editorial.col.comment") }}
                                    </th>
                                    <th
                                        class="border-border-1 border-b px-2 py-2 text-center font-normal"
                                    >
                                        {{ t("editorial.col.publish") }}
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
                                    :key="row.id || `new-${i}`"
                                    class="[&>td]:border-border-1/50 transition-colors [&>td]:border-b"
                                    :class="
                                        i === activeIndex
                                            ? 'bg-primary-default/10'
                                            : ''
                                    "
                                >
                                    <td class="py-2 pl-2">
                                        <DesignButton
                                            variant="tertiary"
                                            size="small"
                                            icon="tabler:player-play"
                                            :disabled="!previewUrl"
                                            @click="preview(row)"
                                        />
                                    </td>
                                    <td class="py-2 pr-2 pl-3">
                                        <DesignInput
                                            v-if="effectiveMode === 'edit'"
                                            v-model="row.name"
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
                                        class="text-body-3 text-text-muted px-2 py-2 tabular-nums"
                                    >
                                        {{ durationOf(row) }}
                                    </td>
                                    <td class="px-2 py-2">
                                        <DesignInput
                                            v-if="effectiveMode === 'edit'"
                                            v-model="row.comment"
                                        />
                                        <span
                                            v-else
                                            class="text-body-3 text-text-muted"
                                        >
                                            {{ row.comment || "—" }}
                                        </span>
                                    </td>
                                    <td class="px-2 py-2">
                                        <div class="flex justify-center">
                                            <DesignSwitch
                                                :model-value="row.publish"
                                                @update:model-value="
                                                    onPublishToggle(row, $event)
                                                "
                                            />
                                        </div>
                                    </td>
                                    <td
                                        v-if="effectiveMode === 'edit'"
                                        class="py-2 pl-2"
                                    >
                                        <div class="flex items-center gap-0.5">
                                            <DesignButton
                                                variant="tertiary"
                                                size="small"
                                                icon="tabler:chevron-up"
                                                :disabled="i === 0"
                                                @click="move(i, -1)"
                                            />
                                            <DesignButton
                                                variant="tertiary"
                                                size="small"
                                                icon="tabler:chevron-down"
                                                :disabled="
                                                    i === rows.length - 1
                                                "
                                                @click="move(i, 1)"
                                            />
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
</template>
