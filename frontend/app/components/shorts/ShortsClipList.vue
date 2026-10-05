<script setup lang="ts">
import type { Clip } from "~/utils/shortsClips";
import type { FlatSegment } from "~/utils/shortsTranscript";

const props = defineProps<{
    clips: Clip[];
    activeId?: string;
    overlapping: Set<string>;
    /** Length of the asset, so timestamps match the rest of the page. */
    duration: number;
    /** Transcript, if there is one, used to label each clip by what it says. */
    segments?: FlatSegment[];
}>();

const emit = defineEmits<{
    activate: [id: string];
    remove: [id: string];
    add: [];
}>();

const { t } = useI18n();

const labels = computed(() => {
    const out = new Map<string, string>();
    for (const clip of props.clips) {
        const label = labelForRange(props.segments ?? [], clip);
        if (label) out.set(clip.id, label);
    }
    return out;
});
</script>

<template>
    <section class="border-border-1 flex flex-col rounded-xl border">
        <header
            class="border-border-1 flex shrink-0 items-center gap-2 border-b px-3 py-2"
        >
            <h2 class="text-title-3 text-text-default">
                {{ t("shorts.generation.clipCount", { count: clips.length }) }}
            </h2>
            <DesignButton
                size="small"
                variant="secondary"
                class="border-border-1 ml-auto shrink-0 border"
                icon="tabler:plus"
                @click="emit('add')"
            >
                {{ t("shorts.generation.addClip") }}
                <span class="text-text-hint ml-1 text-xs">A</span>
            </DesignButton>
        </header>

        <ul class="min-h-0 flex-1 overflow-y-auto p-1.5">
            <li v-for="(clip, index) in clips" :key="clip.id">
                <button
                    type="button"
                    :class="[
                        'flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left',
                        clip.id === activeId
                            ? 'bg-primary-default/15 text-text-default'
                            : 'text-text-muted hover:bg-surface-indent',
                    ]"
                    @click="emit('activate', clip.id)"
                >
                    <span
                        :class="[
                            'w-4 shrink-0 text-center text-xs font-semibold tabular-nums',
                            clip.id === activeId
                                ? 'text-text-default'
                                : 'text-text-hint',
                        ]"
                    >
                        {{ index + 1 }}
                    </span>

                    <span class="min-w-0 flex-1">
                        <span class="block truncate text-sm">
                            {{
                                labels.get(clip.id) ??
                                t("shorts.generation.clipUnlabelled")
                            }}
                        </span>
                        <span class="text-text-hint block text-xs tabular-nums">
                            {{ formatClock(clip.start, duration) }}–{{
                                formatClock(clip.end, duration)
                            }}
                            <span
                                :class="
                                    clipLength(clip) > 60
                                        ? 'text-semantic-warning'
                                        : ''
                                "
                            >
                                · {{ formatClock(clipLength(clip)) }}
                            </span>
                        </span>
                    </span>

                    <DesignTooltip
                        v-if="overlapping.has(clip.id)"
                        :content="t('shorts.generation.clipOverlaps')"
                    >
                        <Icon
                            name="tabler:layers-intersect"
                            class="text-semantic-warning size-4 shrink-0"
                        />
                    </DesignTooltip>

                    <Icon
                        v-if="clips.length > 1"
                        name="tabler:x"
                        class="text-text-hint hover:text-semantic-error size-4 shrink-0"
                        :aria-label="t('shorts.generation.removeClip')"
                        @click.stop="emit('remove', clip.id)"
                    />
                </button>
            </li>
        </ul>
    </section>
</template>
