<script lang="ts" setup>
const props = defineProps<{
    /** The reference as the editor typed it, e.g. "1Kor 1,18+23-24". */
    reference: string;
}>();

const { t } = useI18n();
const api = useAPI();

type Lookup = {
    reference: string;
    translation: string;
    verses: { number: number; text: string }[];
};

// Verse text never changes, so one lookup per reference serves the whole page.
const cache = useState<Record<string, Lookup>>("bible-verses", () => ({}));

const loading = ref(false);
const failed = ref(false);
const result = computed(() => cache.value[props.reference]);

// Fetched when the pointer (or focus) arrives, so the text is usually there by
// the time the tooltip has finished opening.
async function load() {
    if (result.value || loading.value) return;
    loading.value = true;
    failed.value = false;
    try {
        const res = await api.getBibleVerses({ reference: props.reference });
        cache.value[props.reference] = {
            reference: res.reference,
            translation: res.translation,
            verses: res.verses.map((v) => ({ number: v.number, text: v.text })),
        };
    } catch {
        failed.value = true;
    } finally {
        loading.value = false;
    }
}
</script>

<template>
    <DesignTooltip
        size="panel"
        placement="bottom-start"
        interactive
        :open-delay="200"
        :close-delay="200"
    >
        <button
            type="button"
            class="bg-text-default/10 hover:bg-text-default/20 dark:bg-surface-raise dark:hover:bg-text-default/25 gradient-border text-body-2 text-text-default ds-focus-ring cursor-default rounded-lg px-3 py-1.5 transition-colors"
            @mouseenter="load"
            @focusin="load"
        >
            {{ reference }}
        </button>

        <template #content>
            <!-- Teleported out of the review view, so it needs the scale too. -->
            <div class="type-scale-lg">
                <div class="flex items-baseline justify-between gap-4">
                    <p class="text-title-1 text-text-default">
                        {{ result?.reference || reference }}
                    </p>
                    <span
                        v-if="result?.translation"
                        class="text-body-3 text-text-muted shrink-0"
                    >
                        {{ result.translation }}
                    </span>
                </div>

                <div class="mt-3">
                    <p
                        v-if="loading"
                        class="text-body-2 text-text-hint flex items-center gap-2"
                    >
                        <Icon name="svg-spinners:ring-resize" class="size-4" />
                        {{ t("editorial.verseLoading") }}
                    </p>
                    <p
                        v-else-if="failed"
                        class="text-body-2 text-semantic-error"
                    >
                        {{ t("editorial.verseFailed") }}
                    </p>
                    <p
                        v-else-if="!result?.verses.length"
                        class="text-body-2 text-text-hint"
                    >
                        {{ t("editorial.verseNotFound") }}
                    </p>
                    <p
                        v-else
                        class="text-body-1 text-text-default leading-relaxed"
                    >
                        <span
                            v-for="verse in result.verses"
                            :key="verse.number"
                        >
                            <sup class="text-text-hint mr-0.5 tabular-nums">
                                {{ verse.number }}
                            </sup>
                            {{ verse.text }}
                        </span>
                    </p>
                </div>
            </div>
        </template>
    </DesignTooltip>
</template>
