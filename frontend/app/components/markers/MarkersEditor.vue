<script setup lang="ts">
import {
    MARKER_TYPES,
    formatMarkerDuration,
    formatMarkerTime,
    markerTypeMeta,
    parseMarkerTime,
} from "~/utils/markers";
import type { Marker } from "~/utils/markers";

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
const note = computed({
    get: () => props.marker?.note ?? "",
    set: (v) => emit("update", { note: v }),
});

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

const labelInput = useTemplateRef("labelInput");
function focusLabel() {
    const root = labelInput.value?.$el as HTMLElement | undefined;
    root?.querySelector("input")?.focus();
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

        <DesignInput
            ref="labelInput"
            v-model="label"
            :label="t('markers.editor.label')"
            :placeholder="t('markers.editor.labelPlaceholder')"
        />

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
