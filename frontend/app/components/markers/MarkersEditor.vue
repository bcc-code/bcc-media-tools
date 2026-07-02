<script setup lang="ts">
import {
    MARKER_TYPES,
    formatMarkerDuration,
    formatMarkerTime,
    isMarkerUnresolved,
    markerTypeMeta,
    parseMarkerTime,
} from "~/utils/markers";
import type { Marker } from "~/utils/markers";
import { TYPE_TO_PB } from "~/composables/useMarkers";
import type { ComboboxItem } from "~/components/design/DesignCombobox.vue";
import type { Entity } from "~~/src/gen/api/v1/api_pb";

const props = defineProps<{
    marker: Marker | undefined;
    currentTime: number;
}>();

const emit = defineEmits<{
    update: [patch: Partial<Omit<Marker, "id">>];
    remove: [];
    seek: [seconds: number];
    preview: [start: number, end: number];
}>();

const { t } = useI18n();

const NUDGE_STEP = 1;

const iconBtnClass =
    "ds-focus-ring text-text-muted hover:bg-surface-indent hover:text-text-default flex items-center justify-center rounded-lg p-1.5 transition-colors";

const typeItems = computed(() =>
    MARKER_TYPES.map((m) => ({
        label: t(`markers.types.${m.value}`),
        value: m.value,
    })),
);

const type = computed({
    get: () => props.marker?.type ?? "name-super",
    set: (v) => emit("update", { type: v as Marker["type"] }),
});
const label = computed({
    get: () => props.marker?.label ?? "",
    set: (v) => emit("update", { label: v }),
});

// Bible-verse labels autocomplete against the bible server. Selecting a
// suggestion links the marker to a canonical reference (entityId); typing free
// text clears that link but keeps working as a plain label.
const api = useAPI();
// Raw suggestions kept so a selection can recover the entity's `source` (which
// registry it came from) rather than re-guessing it on the frontend.
const entities = ref<Entity[]>([]);
const entityItems = computed<ComboboxItem[]>(() =>
    entities.value.map((e) => ({
        value: e.id,
        label: e.label,
        detail: e.detail,
    })),
);
const entityLoading = ref(false);
let searchSeq = 0;

function resetSuggestions() {
    entities.value = [];
    entityLoading.value = false;
}

// Drop suggestions (and invalidate any in-flight search) when switching markers.
watch(
    () => props.marker?.id,
    () => {
        resetSuggestions();
        searchSeq++;
    },
);

async function onEntitySearch(query: string) {
    const q = query.trim();
    if (!q) {
        resetSuggestions();
        return;
    }
    const seq = ++searchSeq;
    entityLoading.value = true;
    try {
        const res = await api.searchEntities({
            type: TYPE_TO_PB[type.value],
            query: q,
        });
        // Ignore results from a stale request that resolved out of order.
        if (seq !== searchSeq) return;
        entities.value = res.entities;
    } catch (err) {
        if (seq !== searchSeq) return;
        console.error("Entity search failed", err);
        entities.value = [];
    } finally {
        if (seq === searchSeq) entityLoading.value = false;
    }
}

// Free-text edit: keep the label, drop any stale entity link.
function onLabelInput(v: string) {
    emit("update", { label: v, entityId: undefined, entitySource: undefined });
}

function onEntitySelect(item: ComboboxItem) {
    const source =
        entities.value.find((e) => e.id === item.value)?.source ?? "";
    emit("update", {
        label: item.label,
        entityId: item.value,
        entitySource: source,
    });
    entities.value = [];
}

// Try to auto-resolve just this marker's label. Links it on a confident match;
// otherwise nudges the user to pick from the suggestions manually.
const toaster = useDesignToaster();
const resolving = ref(false);
async function resolveThis() {
    const marker = props.marker;
    if (!marker) return;
    resolving.value = true;
    try {
        const res = await api.resolveReferences({
            queries: [
                {
                    refId: marker.id,
                    type: TYPE_TO_PB[marker.type],
                    text: marker.label,
                },
            ],
        });
        const result = res.results[0];
        if (result?.resolved && result.entity) {
            emit("update", {
                label: result.entity.label,
                entityId: result.entity.id,
                entitySource: result.entity.source,
            });
        } else {
            toaster.create({
                title: t("markers.resolve.noMatch"),
                type: "warning",
            });
        }
    } catch (err) {
        console.error("Resolve failed", err);
        toaster.create({ title: t("markers.resolve.error"), type: "error" });
    } finally {
        resolving.value = false;
    }
}
const note = computed({
    get: () => props.marker?.note ?? "",
    set: (v) => emit("update", { note: v }),
});

const unresolved = computed(
    () => !!props.marker && isMarkerUnresolved(props.marker),
);

const startStr = ref("");
const endStr = ref("");
watch(
    () => props.marker,
    (m) => {
        if (!m) return;
        startStr.value = formatMarkerTime(m.start);
        endStr.value = formatMarkerTime(m.end);
    },
    { immediate: true, deep: true },
);

function commit(which: "start" | "end") {
    const raw = which === "start" ? startStr.value : endStr.value;
    const seconds = parseMarkerTime(raw);
    if (!Number.isFinite(seconds) || !props.marker) {
        // Revert the field to the last good value.
        if (props.marker) {
            startStr.value = formatMarkerTime(props.marker.start);
            endStr.value = formatMarkerTime(props.marker.end);
        }
        return;
    }
    // Clamp so the range can't invert (Out before In or vice-versa).
    const value =
        which === "start"
            ? Math.min(Math.max(0, seconds), props.marker.end)
            : Math.max(seconds, props.marker.start);
    emit("update", { [which]: value });
    if (which === "start") startStr.value = formatMarkerTime(value);
    else endStr.value = formatMarkerTime(value);
}

function setToCurrent(which: "start" | "end") {
    emit("update", { [which]: Math.round(props.currentTime) });
}

function nudge(which: "start" | "end", delta: number) {
    if (!props.marker) return;
    const base = which === "start" ? props.marker.start : props.marker.end;
    let value = Math.round(base) + delta;
    if (which === "start")
        value = Math.min(Math.max(0, value), props.marker.end);
    else value = Math.max(value, props.marker.start);
    emit("update", { [which]: value });
}

const duration = computed(() => {
    if (!props.marker) return 0;
    return Math.max(0, props.marker.end - props.marker.start);
});

const labelInput = useTemplateRef<HTMLElement>("labelInput");
function focusLabel() {
    labelInput.value?.querySelector("input")?.focus();
}
defineExpose({ focusLabel });
</script>

<template>
    <div
        v-if="marker"
        class="gradient-border bg-surface-default shadow-resting flex flex-col gap-4 rounded-2xl p-4"
    >
        <div class="flex items-center justify-between">
            <h2 class="text-title-2 text-text-default flex items-center gap-2">
                <Icon :name="markerTypeMeta(marker.type).icon" class="size-5" />
                {{ t("markers.editor.title") }}
            </h2>
            <DesignBadge
                :variant="marker.source === 'imported' ? 'info' : 'neutral'"
            >
                {{ t(`markers.source.${marker.source}`) }}
            </DesignBadge>
        </div>

        <div>
            <span class="text-body-3 text-text-muted mb-1 block">
                {{ t("markers.editor.type") }}
            </span>
            <div class="flex items-center gap-2">
                <Icon
                    :name="markerTypeMeta(type).icon"
                    class="size-5 shrink-0"
                    :class="markerTypeMeta(type).iconColor"
                />
                <DesignSelect
                    v-model="type"
                    :items="typeItems"
                    class="flex-1"
                />
            </div>
        </div>

        <div ref="labelInput">
            <DesignCombobox
                v-if="type === 'bible-verse'"
                :model-value="label"
                :items="entityItems"
                :loading="entityLoading"
                :field-label="t('markers.editor.label')"
                :placeholder="t('markers.editor.biblePlaceholder')"
                :empty-text="t('markers.editor.bibleEmpty')"
                leading-icon="tabler:book-2"
                @update:model-value="onLabelInput"
                @search="onEntitySearch"
                @select="onEntitySelect"
            >
                <template #trailing>
                    <DesignTooltip
                        v-if="unresolved && !entityLoading"
                        :content="t('markers.resolve.one')"
                    >
                        <button
                            type="button"
                            :disabled="resolving"
                            class="ds-focus-ring text-text-muted hover:text-text-default flex items-center justify-center rounded-md p-1 transition-colors disabled:opacity-50"
                            @mousedown.prevent
                            @click.stop="resolveThis"
                        >
                            <Icon
                                :name="
                                    resolving
                                        ? 'tabler:loader-2'
                                        : 'tabler:wand'
                                "
                                class="size-4"
                                :class="{ 'animate-spin': resolving }"
                            />
                        </button>
                    </DesignTooltip>
                </template>
            </DesignCombobox>
            <DesignInput
                v-else
                v-model="label"
                :label="t('markers.editor.label')"
                :placeholder="t('markers.editor.labelPlaceholder')"
            />
            <p
                v-if="unresolved"
                class="text-caption-1 text-semantic-warning mt-1.5 flex items-center gap-1"
            >
                <Icon name="tabler:alert-triangle" class="size-3.5 shrink-0" />
                {{ t("markers.review.hint") }}
            </p>
        </div>

        <DesignTextarea
            v-model="note"
            :label="t('markers.editor.note')"
            :rows="2"
            :placeholder="t('markers.editor.notePlaceholder')"
        />

        <div>
            <div class="mb-2 flex items-center justify-between">
                <span class="text-body-3 text-text-muted">
                    {{ t("markers.editor.timing") }}
                </span>
                <span class="text-text-hint text-caption-1 tabular-nums">
                    {{ formatMarkerDuration(duration) }}
                </span>
            </div>

            <div class="flex flex-col gap-2">
                <div class="flex items-center gap-2">
                    <span class="text-text-hint text-caption-1 w-6 shrink-0">
                        {{ t("markers.editor.in") }}
                    </span>
                    <DesignInput
                        v-model="startStr"
                        class="w-28 shrink-0 tabular-nums"
                        @change="commit('start')"
                    />
                    <div class="ml-auto flex items-center gap-0.5">
                        <DesignTooltip
                            :content="t('markers.editor.setToPlayhead')"
                        >
                            <button
                                type="button"
                                :class="iconBtnClass"
                                @click="setToCurrent('start')"
                            >
                                <Icon
                                    name="tabler:arrow-bar-to-left"
                                    class="size-4"
                                />
                            </button>
                        </DesignTooltip>
                        <DesignTooltip :content="t('markers.editor.nudgeBack')">
                            <button
                                type="button"
                                :class="iconBtnClass"
                                @click="nudge('start', -NUDGE_STEP)"
                            >
                                <Icon name="tabler:minus" class="size-4" />
                            </button>
                        </DesignTooltip>
                        <DesignTooltip
                            :content="t('markers.editor.nudgeForward')"
                        >
                            <button
                                type="button"
                                :class="iconBtnClass"
                                @click="nudge('start', NUDGE_STEP)"
                            >
                                <Icon name="tabler:plus" class="size-4" />
                            </button>
                        </DesignTooltip>
                        <DesignTooltip :content="t('markers.editor.seekTo')">
                            <button
                                type="button"
                                :class="iconBtnClass"
                                @click="emit('seek', marker.start)"
                            >
                                <Icon
                                    name="tabler:player-play"
                                    class="size-4"
                                />
                            </button>
                        </DesignTooltip>
                    </div>
                </div>

                <div class="flex items-center gap-2">
                    <span class="text-text-hint text-caption-1 w-6 shrink-0">
                        {{ t("markers.editor.out") }}
                    </span>
                    <DesignInput
                        v-model="endStr"
                        class="w-28 shrink-0 tabular-nums"
                        @change="commit('end')"
                    />
                    <div class="ml-auto flex items-center gap-0.5">
                        <DesignTooltip
                            :content="t('markers.editor.setToPlayhead')"
                        >
                            <button
                                type="button"
                                :class="iconBtnClass"
                                @click="setToCurrent('end')"
                            >
                                <Icon
                                    name="tabler:arrow-bar-to-right"
                                    class="size-4"
                                />
                            </button>
                        </DesignTooltip>
                        <DesignTooltip :content="t('markers.editor.nudgeBack')">
                            <button
                                type="button"
                                :class="iconBtnClass"
                                @click="nudge('end', -NUDGE_STEP)"
                            >
                                <Icon name="tabler:minus" class="size-4" />
                            </button>
                        </DesignTooltip>
                        <DesignTooltip
                            :content="t('markers.editor.nudgeForward')"
                        >
                            <button
                                type="button"
                                :class="iconBtnClass"
                                @click="nudge('end', NUDGE_STEP)"
                            >
                                <Icon name="tabler:plus" class="size-4" />
                            </button>
                        </DesignTooltip>
                        <DesignTooltip :content="t('markers.editor.seekTo')">
                            <button
                                type="button"
                                :class="iconBtnClass"
                                @click="emit('seek', marker.end)"
                            >
                                <Icon
                                    name="tabler:player-play"
                                    class="size-4"
                                />
                            </button>
                        </DesignTooltip>
                    </div>
                </div>
            </div>

            <div class="mt-3 flex justify-center">
                <DesignButton
                    size="small"
                    variant="tertiary"
                    icon="tabler:player-play"
                    @click="emit('preview', marker.start, marker.end)"
                >
                    {{ t("markers.editor.previewRange") }}
                </DesignButton>
            </div>
        </div>

        <DesignButton
            variant="secondary"
            intent="danger"
            icon="tabler:trash"
            class="mt-auto"
            @click="emit('remove')"
        >
            {{ t("markers.editor.remove") }}
        </DesignButton>
    </div>

    <div
        v-else
        class="border-border-1 text-text-hint flex flex-col items-center justify-center gap-2 rounded-xl border border-dashed p-8 text-center text-sm"
    >
        <Icon name="tabler:click" class="size-6" />
        {{ t("markers.editor.noSelection") }}
    </div>
</template>
