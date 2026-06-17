<script lang="ts" setup>
import type { VaultItem, VaultFacet } from "~~/src/gen/api/v1/api_pb";

const api = useAPI();
const { t } = useI18n();
const config = useRuntimeConfig();
const base = config.public.grpcUrl;

useHead({ title: "Vault" });

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({ id: "vault", title: "vault" });
});

const MEDIA_CATEGORIES = ["video", "audio", "image", "other"] as const;
type MediaCategory = (typeof MEDIA_CATEGORIES)[number];

const query = ref("");
const selectedTypes = ref<MediaCategory[]>([]);
const page = ref(1);

const categoryItems = computed(() =>
    MEDIA_CATEGORIES.map((value) => ({
        value,
        label: t(`vault.type.${value}`),
    })),
);

const items = ref<VaultItem[]>([]);
const facets = ref<VaultFacet[]>([]);
const totalHits = ref(0);
const pageSize = ref(50);
const loading = ref(false);
const loaded = ref(false);

async function load() {
    loading.value = true;
    try {
        const res = await api.vaultSearch({
            query: query.value,
            mediaTypes: selectedTypes.value,
            page: page.value,
        });
        items.value = res.items;
        facets.value = res.facets;
        totalHits.value = res.totalHits;
        if (res.pageSize) pageSize.value = res.pageSize;
    } finally {
        loading.value = false;
        loaded.value = true;
    }
}

// Debounce free-text typing; filter/page changes apply immediately.
watchDebounced(
    query,
    () => {
        page.value = 1;
        load();
    },
    { debounce: 300 },
);
watch(selectedTypes, () => {
    page.value = 1;
    load();
});
watch(page, load);
onMounted(load);

function facetCount(type: string): number {
    return facets.value.find((f) => f.mediaType === type)?.count ?? 0;
}

const hasFilter = computed(() => selectedTypes.value.length > 0);
function clearFilters() {
    selectedTypes.value = [];
}

const totalPages = computed(() =>
    Math.max(1, Math.ceil(totalHits.value / pageSize.value)),
);
const rangeFrom = computed(() =>
    totalHits.value === 0 ? 0 : (page.value - 1) * pageSize.value + 1,
);
const rangeTo = computed(() =>
    Math.min(page.value * pageSize.value, totalHits.value),
);
</script>

<template>
    <div>
        <!-- Search bar -->
        <div class="border-default border-b px-6 py-5">
            <UInput
                v-model="query"
                icon="tabler:search"
                size="lg"
                :loading="loading"
                :placeholder="t('vault.searchPlaceholder')"
                class="w-full max-w-xl"
            />
        </div>

        <div class="flex items-start">
            <!-- Filter sidebar -->
            <aside
                class="border-default w-64 shrink-0 self-stretch border-r p-6"
            >
                <div class="mb-5 flex items-center justify-between">
                    <h2 class="text-lg font-semibold">
                        {{ t("vault.filters") }}
                    </h2>
                    <button
                        v-if="hasFilter"
                        class="text-muted hover:text-default text-xs"
                        @click="clearFilters"
                    >
                        {{ t("vault.clear") }}
                    </button>
                </div>
                <UCheckboxGroup
                    v-model="selectedTypes"
                    :items="categoryItems"
                    :legend="t('vault.mediaType')"
                    size="lg"
                    :ui="{
                        legend: 'text-muted mb-3 text-[11px] font-semibold tracking-wide uppercase',
                        fieldset: 'gap-y-2',
                    }"
                >
                    <template #label="{ item }">
                        <span
                            class="flex w-full items-center justify-between gap-2"
                        >
                            <span>{{ item.label }}</span>
                            <span class="text-muted font-mono text-xs">
                                {{ facetCount(item.value) }}
                            </span>
                        </span>
                    </template>
                </UCheckboxGroup>
            </aside>

            <!-- Results -->
            <main class="min-w-0 flex-1 p-6">
                <div class="mb-4 flex items-baseline justify-between">
                    <h2 class="text-sm font-semibold">
                        {{ t("vault.results") }}
                    </h2>
                    <span class="text-muted">
                        {{
                            t("vault.resultsRange", {
                                from: rangeFrom,
                                to: rangeTo,
                                total: totalHits,
                            })
                        }}
                    </span>
                </div>

                <!-- Searching state -->
                <div v-if="loading" class="grid grid-cols-5 gap-4">
                    <div
                        v-for="n in 10"
                        :key="n"
                        class="border-default overflow-hidden rounded-[14px] border"
                    >
                        <USkeleton class="aspect-16/10 w-full rounded-none" />
                        <div class="space-y-2 p-3">
                            <USkeleton class="h-3 w-3/4" />
                            <USkeleton class="h-2 w-1/2" />
                        </div>
                    </div>
                </div>

                <div v-else-if="items.length" class="grid grid-cols-5 gap-4">
                    <VaultCard
                        v-for="item in items"
                        :key="item.VXID"
                        :item="item"
                        :base="base"
                    />
                </div>

                <div v-else-if="loaded" class="text-muted py-20 text-center">
                    <UIcon name="i-lucide-search" class="size-10 opacity-50" />
                    <p class="mt-3 text-sm">{{ t("vault.noResults") }}</p>
                </div>

                <!-- Pagination -->
                <UPagination
                    v-if="totalPages > 1"
                    v-model:page="page"
                    :total="totalHits"
                    :items-per-page="pageSize"
                    :sibling-count="1"
                    show-edges
                    :disabled="loading"
                    class="mt-6 flex justify-center"
                />
            </main>
        </div>
    </div>
</template>
