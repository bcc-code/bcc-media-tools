<script setup lang="ts">
import type { PlayoutEvent } from "~~/src/gen/api/v1/api_pb";

// Searchable list of Playout events, shared by the Playout import dialog and
// the "new live meeting" form.
const props = defineProps<{
    events: PlayoutEvent[];
    loading?: boolean;
    selectedId?: string;
}>();

const emit = defineEmits<{ select: [event: PlayoutEvent] }>();

const { t } = useI18n();

const search = ref("");
const filtered = computed(() => {
    const q = search.value.trim().toLowerCase();
    if (!q) return props.events;
    return props.events.filter((e) => e.name.toLowerCase().includes(q));
});
</script>

<template>
    <div class="flex flex-col gap-3">
        <DesignInput
            v-model="search"
            leading-icon="tabler:search"
            :placeholder="t('editorial.searchPlayoutEvents')"
        />

        <div v-if="loading" class="flex flex-col gap-2">
            <DesignSkeleton v-for="i in 4" :key="i" class="h-14 rounded-xl" />
        </div>

        <p
            v-else-if="filtered.length === 0"
            class="text-body-3 text-text-hint py-8 text-center"
        >
            {{ t("editorial.noPlayoutEvents") }}
        </p>

        <ul v-else class="flex max-h-96 flex-col gap-2 overflow-y-auto">
            <li v-for="e in filtered" :key="e.id">
                <button
                    type="button"
                    class="gradient-border shadow-resting ds-focus-ring hover:bg-surface-indent flex w-full items-center justify-between gap-4 rounded-xl px-4 py-2.5 text-left transition-colors"
                    :class="
                        e.id === selectedId
                            ? 'bg-primary-default/10'
                            : 'bg-surface-raise'
                    "
                    :aria-pressed="
                        selectedId === undefined
                            ? undefined
                            : e.id === selectedId
                    "
                    @click="emit('select', e)"
                >
                    <div class="min-w-0">
                        <p class="text-title-2 text-text-default truncate">
                            {{ e.name || e.id }}
                        </p>
                        <p class="text-caption-1 text-text-hint truncate">
                            {{
                                [e.date, e.status, e.productionUnit]
                                    .filter(Boolean)
                                    .join(" · ")
                            }}
                        </p>
                    </div>
                    <Icon
                        v-if="e.id === selectedId"
                        name="tabler:check"
                        class="text-primary-default size-5 shrink-0"
                    />
                </button>
            </li>
        </ul>
    </div>
</template>
