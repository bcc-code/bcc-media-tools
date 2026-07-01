<script setup lang="ts">
import { MARKER_TYPES, markerTypeMeta } from "~/utils/markers";
import type { Marker } from "~/utils/markers";

const props = defineProps<{
    marker: Marker | undefined;
    currentTime: number;
}>();

const emit = defineEmits<{
    update: [patch: Partial<Omit<Marker, "id">>];
    remove: [];
    seek: [seconds: number];
}>();

const { t } = useI18n();

const typeItems = computed(() =>
    MARKER_TYPES.map((m) => ({
        label: t(`markers.types.${m.value}`),
        value: m.value,
    })),
);

// Two-way bridges that emit a patch on every change.
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

// In/out are edited as formatted strings, committed (parsed) on change so a
// half-typed value never corrupts the marker.
const startStr = ref("");
const endStr = ref("");
watch(
    () => props.marker,
    (m) => {
        if (!m) return;
        startStr.value = formatTime(m.start);
        endStr.value = formatTime(m.end);
    },
    { immediate: true, deep: true },
);

function commit(which: "start" | "end") {
    const raw = which === "start" ? startStr.value : endStr.value;
    try {
        const seconds = secondsFromFormattedTime(raw);
        if (!Number.isFinite(seconds) || seconds < 0)
            throw new Error("bad time");
        emit("update", { [which]: seconds });
    } catch {
        // Revert the field to the last good value.
        if (!props.marker) return;
        startStr.value = formatTime(props.marker.start);
        endStr.value = formatTime(props.marker.end);
    }
}

function setToCurrent(which: "start" | "end") {
    emit("update", { [which]: props.currentTime });
}

const duration = computed(() => {
    if (!props.marker) return 0;
    return Math.max(0, props.marker.end - props.marker.start);
});
</script>

<template>
    <div
        v-if="marker"
        class="border-border-1 bg-surface-default flex flex-col gap-4 rounded-xl border p-4"
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
            <DesignSelect v-model="type" :items="typeItems" />
        </div>

        <DesignInput
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

        <div class="grid grid-cols-2 gap-3">
            <div>
                <span class="text-body-3 text-text-muted mb-1 block">
                    {{ t("markers.editor.in") }}
                </span>
                <DesignInput
                    v-model="startStr"
                    class="tabular-nums"
                    @change="commit('start')"
                />
                <div class="mt-1.5 flex gap-1">
                    <DesignButton
                        size="small"
                        variant="tertiary"
                        icon="tabler:player-track-prev"
                        @click="setToCurrent('start')"
                    >
                        {{ t("markers.editor.setToPlayhead") }}
                    </DesignButton>
                    <DesignButton
                        size="small"
                        variant="tertiary"
                        icon="tabler:player-play"
                        :aria-label="t('markers.editor.seekTo')"
                        @click="emit('seek', marker.start)"
                    />
                </div>
            </div>
            <div>
                <span class="text-body-3 text-text-muted mb-1 block">
                    {{ t("markers.editor.out") }}
                </span>
                <DesignInput
                    v-model="endStr"
                    class="tabular-nums"
                    @change="commit('end')"
                />
                <div class="mt-1.5 flex gap-1">
                    <DesignButton
                        size="small"
                        variant="tertiary"
                        icon="tabler:player-track-next"
                        @click="setToCurrent('end')"
                    >
                        {{ t("markers.editor.setToPlayhead") }}
                    </DesignButton>
                    <DesignButton
                        size="small"
                        variant="tertiary"
                        icon="tabler:player-play"
                        :aria-label="t('markers.editor.seekTo')"
                        @click="emit('seek', marker.end)"
                    />
                </div>
            </div>
        </div>

        <p class="text-text-hint text-caption-1">
            {{ t("markers.editor.duration") }}: {{ duration.toFixed(2) }}s
        </p>

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
