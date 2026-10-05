<script setup lang="ts">
import type { Clip } from "~/utils/shortsClips";

const props = defineProps<{
    clips: Clip[];
    activeId?: string;
    overlapping: Set<string>;
}>();

const emit = defineEmits<{
    activate: [id: string];
    remove: [id: string];
    add: [];
}>();

const { t } = useI18n();

const longClips = computed(
    () => props.clips.filter((c) => clipLength(c) > 60).length,
);
</script>

<template>
    <div class="flex items-center gap-2">
        <span class="text-text-hint shrink-0 text-xs">
            {{ t("shorts.generation.clipCount", { count: clips.length }) }}
        </span>

        <div class="flex min-w-0 flex-1 gap-2 overflow-x-auto py-1">
            <button
                v-for="(clip, index) in clips"
                :key="clip.id"
                type="button"
                :class="[
                    'group flex shrink-0 items-center gap-2 rounded-xl border px-2.5 py-1.5 text-xs transition-colors',
                    clip.id === activeId
                        ? 'border-primary-default bg-primary-default/15 text-text-default'
                        : 'border-border-1 text-text-muted hover:bg-surface-indent',
                ]"
                @click="emit('activate', clip.id)"
            >
                <span class="font-semibold tabular-nums">{{ index + 1 }}</span>
                <span class="tabular-nums">
                    {{ formatClock(clip.start) }}–{{ formatClock(clip.end) }}
                </span>
                <span
                    :class="[
                        'tabular-nums',
                        clipLength(clip) > 60
                            ? 'text-semantic-warning'
                            : 'text-text-hint',
                    ]"
                >
                    {{ formatClock(clipLength(clip)) }}
                </span>
                <DesignTooltip
                    v-if="overlapping.has(clip.id)"
                    :content="t('shorts.generation.clipOverlaps')"
                >
                    <Icon
                        name="tabler:layers-intersect"
                        class="text-semantic-warning size-3.5"
                    />
                </DesignTooltip>
                <Icon
                    v-if="clips.length > 1"
                    name="tabler:x"
                    class="text-text-hint hover:text-semantic-error size-3.5"
                    :aria-label="t('shorts.generation.removeClip')"
                    @click.stop="emit('remove', clip.id)"
                />
            </button>
        </div>

        <DesignTooltip :content="t('shorts.generation.addClipHint')">
            <DesignButton
                size="small"
                variant="secondary"
                class="border-border-1 shrink-0 border"
                icon="tabler:plus"
                @click="emit('add')"
            >
                {{ t("shorts.generation.addClip") }}
                <span class="text-text-hint ml-1 text-xs">A</span>
            </DesignButton>
        </DesignTooltip>
    </div>

    <p v-if="longClips" class="text-semantic-warning text-xs">
        {{ t("shorts.generation.clipsTooLong", { count: longClips }) }}
    </p>
</template>
