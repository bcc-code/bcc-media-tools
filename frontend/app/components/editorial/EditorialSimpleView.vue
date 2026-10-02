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
}>();

const emit = defineEmits<{
    /** The row is mutated in place; the previous value allows rollback. */
    fieldChange: [EditorialRow, EditorialField, string];
    publish: [EditorialRow, "bmm" | "bcc", boolean];
}>();

const { t } = useI18n();
const { typeLabel } = useEditorialTypes();

// ── Playback ──────────────────────────────────────────────
const videoEl = useTemplateRef<HTMLVideoElement>("videoEl");
const currentMs = ref(0);
const videoDurationMs = ref(0);
const playing = ref(false);

function onTimeUpdate(e: Event) {
    currentMs.value = (e.target as HTMLVideoElement).currentTime * 1000;
}

function onLoadedMetadata(e: Event) {
    const d = (e.target as HTMLVideoElement).duration;
    videoDurationMs.value = Number.isFinite(d) ? d * 1000 : 0;
}

function seek(ms: number) {
    const el = videoEl.value;
    const clamped = Math.max(0, Math.min(totalMs.value, ms));
    currentMs.value = clamped;
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

const selectedIndex = ref(0);
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
</script>

<template>
    <div class="flex flex-col gap-8">
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
                        class="text-body-2 text-text-hint ml-auto tabular-nums"
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

                <input
                    v-if="canEdit"
                    v-model="selected.name"
                    :placeholder="t('editorial.untitledItem')"
                    class="text-heading-2 text-text-default placeholder:text-text-hint focus:border-border-1 -mx-1 w-full border-b border-transparent bg-transparent px-1 pb-1 outline-none"
                    @focusin="captureBefore(selected.name)"
                    @change="onFieldChange(selected, 'name')"
                />
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
                        @update:model-value="
                            emit('publish', selected, 'bmm', $event)
                        "
                    />
                    <EditorialPublishToggle
                        :model-value="selected.publishBcc"
                        :target="t('editorial.col.publishBcc')"
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
