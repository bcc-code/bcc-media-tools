<script lang="ts" setup>
import type { EditorialField, EditorialRow } from "~/utils/editorial";
import {
    editorialTypeColor,
    formatMs,
    parseTc,
    rowDurationMs,
    splitBibleVerses,
} from "~/utils/editorial";

const props = defineProps<{
    rows: EditorialRow[];
    previewUrl?: string;
    canEdit: boolean;
    /** Where the playhead was when this view was opened. */
    startMs?: number;
}>();

const emit = defineEmits<{
    /** The row is mutated in place; the previous value allows rollback. */
    fieldChange: [EditorialRow, EditorialField, string];
    publish: [EditorialRow, "bmm" | "bcc", boolean];
    /** Playhead moved; the page mirrors it into the URL. */
    position: [number];
}>();

const { t } = useI18n();
const { typeLabel } = useEditorialTypes();

// ── Playback ──────────────────────────────────────────────
const videoEl = useTemplateRef<HTMLVideoElement>("videoEl");
const currentMs = ref(props.startMs ?? 0);
const videoDurationMs = ref(0);
const playing = ref(false);

function onTimeUpdate(e: Event) {
    currentMs.value = (e.target as HTMLVideoElement).currentTime * 1000;
    emit("position", currentMs.value);
}

// The video only accepts a seek once it knows its duration, so the restore
// waits for metadata — and happens once, not on every later metadata event.
let restored = false;
function onLoadedMetadata(e: Event) {
    const el = e.target as HTMLVideoElement;
    const d = el.duration;
    videoDurationMs.value = Number.isFinite(d) ? d * 1000 : 0;
    if (!restored) {
        restored = true;
        if (currentMs.value > 0) seek(currentMs.value);
    }
}

function seek(ms: number) {
    const el = videoEl.value;
    const clamped = Math.max(0, Math.min(totalMs.value, ms));
    currentMs.value = clamped;
    emit("position", clamped);
    if (el) el.currentTime = clamped / 1000;
}

function togglePlay() {
    const el = videoEl.value;
    if (!el) return;
    if (el.paused) void el.play();
    else el.pause();
}

// The marker list may run past the proxy's duration (or the other way around
// for a partially marked programme), so the timeline spans whichever is longer.
const markersEndMs = computed(() =>
    props.rows.reduce((max, r) => Math.max(max, parseTc(r.end)), 0),
);
const totalMs = computed(() =>
    Math.max(1, videoDurationMs.value, markersEndMs.value),
);

// ── Selection ─────────────────────────────────────────────
// The playhead drives selection, so playing through the programme walks the
// detail panel from item to item. In a gap the last item stays selected.
const activeIndex = computed(() =>
    props.rows.findIndex((r) => {
        const start = parseTc(r.start);
        const end = parseTc(r.end);
        return end > start && currentMs.value >= start && currentMs.value < end;
    }),
);

const selectedIndex = ref(Math.max(0, activeIndex.value));
watch(activeIndex, (i) => {
    if (i >= 0) selectedIndex.value = i;
});
watch(
    () => props.rows.length,
    (n) => {
        if (selectedIndex.value > n - 1)
            selectedIndex.value = Math.max(0, n - 1);
    },
);

const selected = computed<EditorialRow | undefined>(
    () => props.rows[selectedIndex.value],
);

function select(i: number) {
    selectedIndex.value = i;
    const row = props.rows[i];
    if (row) seek(parseTc(row.start));
}

function step(delta: number) {
    const next = selectedIndex.value + delta;
    if (next >= 0 && next < props.rows.length) select(next);
}

// ── Position within the selected item ─────────────────────
const selectedStartMs = computed(() =>
    selected.value ? parseTc(selected.value.start) : 0,
);
const selectedDurationMs = computed(() =>
    selected.value ? rowDurationMs(selected.value) : 0,
);
const itemElapsedMs = computed<number>({
    get() {
        return Math.min(
            selectedDurationMs.value,
            Math.max(0, currentMs.value - selectedStartMs.value),
        );
    },
    set(v) {
        seek(selectedStartMs.value + v);
    },
});

// Only needed for the read-only rendering; editors get the chip editor.
const verses = computed(() =>
    selected.value ? splitBibleVerses(selected.value.bibleVerses) : [],
);

const selectedColor = computed(() =>
    selected.value ? editorialTypeColor(selected.value.type) : undefined,
);

// ── Inline edits ──────────────────────────────────────────
// The review view writes the title and the comment (and the publish flags);
// everything else is read-only here and edited in the table view. There is no
// Save button, so each field persists on change — the value held before the
// edit lets the page roll back a failed write.
const beforeEdit = ref("");

function captureBefore(value: string) {
    beforeEdit.value = value;
}

function onFieldChange(row: EditorialRow, field: EditorialField) {
    emit("fieldChange", row, field, beforeEdit.value);
}

// Titles run long and have to wrap, so the field is a textarea grown to fit its
// content — on typing, and whenever the selected item changes under it.
const titleEl = useTemplateRef<HTMLTextAreaElement>("titleEl");

function growTitle() {
    const el = titleEl.value;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${el.scrollHeight}px`;
}

watch(
    () => selected.value?.name,
    () => nextTick(growTitle),
    { immediate: true },
);
</script>

<template>
    <!-- type-scale-lg enlarges every text utility inside this view. -->
    <div class="type-scale-lg flex flex-col gap-8">
        <div
            class="grid items-start gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,28rem)]"
        >
            <!-- Player -->
            <div>
                <div
                    class="bg-surface-indent relative aspect-video overflow-hidden rounded-2xl"
                >
                    <video
                        v-if="previewUrl"
                        ref="videoEl"
                        :src="previewUrl"
                        class="h-full w-full"
                        @click="togglePlay"
                        @timeupdate="onTimeUpdate"
                        @loadedmetadata="onLoadedMetadata"
                        @play="playing = true"
                        @pause="playing = false"
                    />
                    <div
                        v-else
                        class="text-text-hint flex h-full items-center justify-center"
                    >
                        <Icon name="tabler:video-off" class="size-8" />
                    </div>

                    <!-- Controls, scoped to the selected item. -->
                    <div
                        v-if="previewUrl"
                        class="pointer-events-none absolute inset-x-0 bottom-0 flex items-center gap-4 bg-gradient-to-t from-black/70 to-transparent px-4 pt-10 pb-4"
                    >
                        <button
                            type="button"
                            class="ds-focus-ring pointer-events-auto flex size-14 shrink-0 cursor-pointer items-center justify-center rounded-full bg-white transition-transform duration-200 active:scale-95"
                            :aria-label="
                                playing
                                    ? t('editorial.pause')
                                    : t('editorial.play')
                            "
                            @click="togglePlay"
                        >
                            <Icon
                                :name="
                                    playing
                                        ? 'tabler:player-pause-filled'
                                        : 'tabler:player-play-filled'
                                "
                                class="size-6 text-black"
                            />
                        </button>
                        <DesignSlider
                            v-model="itemElapsedMs"
                            :min="0"
                            :max="Math.max(1, selectedDurationMs)"
                            :step="100"
                            :disabled="!selected"
                            variant="overlay"
                            class="pointer-events-auto"
                        />
                        <span
                            class="text-title-2 text-text-light-default shrink-0 tabular-nums"
                        >
                            {{ formatMs(itemElapsedMs) }} /
                            {{ formatMs(selectedDurationMs) }}
                        </span>
                    </div>
                </div>
            </div>

            <!-- Detail panel -->
            <div v-if="selected" class="flex flex-col gap-5">
                <div class="flex items-center gap-3">
                    <span
                        class="size-4 shrink-0 rounded-full"
                        :style="{ backgroundColor: selectedColor }"
                    />
                    <span class="text-title-1 text-text-default">
                        {{ typeLabel(selected.type) || "—" }}
                    </span>
                    <span
                        class="text-body-2 text-text-muted ml-auto tabular-nums"
                    >
                        {{
                            t("editorial.itemOf", {
                                n: selectedIndex + 1,
                                total: rows.length,
                            })
                        }}
                    </span>
                    <div class="flex items-center gap-1">
                        <DesignButton
                            variant="tertiary"
                            icon="tabler:chevron-up"
                            :disabled="selectedIndex === 0"
                            :aria-label="t('editorial.previousItem')"
                            @click="step(-1)"
                        />
                        <DesignButton
                            variant="tertiary"
                            icon="tabler:chevron-down"
                            :disabled="selectedIndex >= rows.length - 1"
                            :aria-label="t('editorial.nextItem')"
                            @click="step(1)"
                        />
                    </div>
                </div>

                <!-- The title is the one thing reviewers rename, so the
                     field has to read as editable without becoming a box:
                     a standing underline, a tint on hover, and a pencil. -->
                <div v-if="canEdit" class="group relative -mx-1">
                    <textarea
                        ref="titleEl"
                        v-model="selected.name"
                        rows="1"
                        :aria-label="t('editorial.col.title')"
                        :placeholder="t('editorial.untitledItem')"
                        class="text-heading-2 text-text-default placeholder:text-text-hint border-border-1 hover:bg-surface-indent focus:border-text-hint dark:hover:bg-text-default/5 w-full resize-none overflow-hidden rounded-t-lg border-b bg-transparent px-1 pt-1 pr-10 pb-1.5 transition-colors outline-none"
                        @input="growTitle"
                        @keydown.enter.prevent="titleEl?.blur()"
                        @focusin="captureBefore(selected.name)"
                        @change="onFieldChange(selected, 'name')"
                    />
                    <Icon
                        name="tabler:pencil"
                        class="text-text-hint pointer-events-none absolute top-2 right-2 size-5 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100"
                    />
                </div>
                <h2 v-else class="text-heading-2 text-text-default">
                    {{ selected.name || t("editorial.untitledItem") }}
                </h2>

                <div class="flex flex-col gap-3">
                    <p
                        v-if="selected.contributors"
                        class="text-title-1 text-text-muted"
                    >
                        {{ selected.contributors }}
                    </p>

                    <div v-if="verses.length" class="flex flex-wrap gap-2">
                        <EditorialVerseChip
                            v-for="v in verses"
                            :key="v"
                            :reference="v"
                        />
                    </div>
                </div>

                <!-- Comment and the publish flags stay open to reviewers
                     without edit rights: they are what the review is for. -->
                <DesignTextarea
                    v-model="selected.comment"
                    :label="t('editorial.col.comment')"
                    :rows="3"
                    size="large"
                    @focusin="captureBefore(selected.comment)"
                    @change="onFieldChange(selected, 'comment')"
                />

                <div class="flex flex-col gap-2">
                    <EditorialPublishToggle
                        :model-value="selected.publishBmm"
                        :target="t('editorial.col.publishBmm')"
                        logo="/images/logos/bmm.png"
                        @update:model-value="
                            emit('publish', selected, 'bmm', $event)
                        "
                    />
                    <EditorialPublishToggle
                        :model-value="selected.publishBcc"
                        :target="t('editorial.col.publishBcc')"
                        logo="/images/logos/bcc-media.png"
                        @update:model-value="
                            emit('publish', selected, 'bcc', $event)
                        "
                    />
                </div>
            </div>
        </div>

        <EditorialTimeline
            :rows="rows"
            :selected-index="selectedIndex"
            :current-ms="currentMs"
            :total-ms="totalMs"
            @select="select"
            @seek="seek"
        />
    </div>
</template>
