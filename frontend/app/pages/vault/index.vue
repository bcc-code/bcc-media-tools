<script lang="ts" setup>
import type { VaultItem, VaultFacet } from "~~/src/gen/api/v1/api_pb";

const api = useAPI();
const { t } = useI18n();
const { formatNumber } = useNumberFormat();
const config = useRuntimeConfig();
const base = config.public.grpcUrl;

useHead({ title: "Vault" });

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({ id: "vault", title: "vault" });
});

const MEDIA_CATEGORIES = ["video", "audio", "image"] as const;

const query = useQueryRef("q", "");
const selectedTypes = useQueryRef<string[]>("types", []);
const page = useQueryRef("page", 1);

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

const hasFilter = computed(() => selectedTypes.value.length > 0);
function clearFilters() {
    selectedTypes.value = [];
}

const isPhone = useIsPhone();
const filtersOpen = ref(false);

// Shared by the skeleton and the results grid so the two cannot drift apart.
// 320px rather than 400 so tablets still get two columns once the sidebar goes.
const gridClass =
    "grid grid-cols-1 gap-4 sm:grid-cols-[repeat(auto-fit,minmax(320px,1fr))]";

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
        <div
            class="border-border-1 flex flex-col gap-3 border-b px-4 py-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4 sm:px-6 sm:py-5"
        >
            <DesignInput
                v-model="query"
                leading-icon="tabler:search"
                :placeholder="t('vault.searchPlaceholder')"
                class="w-full sm:max-w-xl"
            >
                <template #trailing>
                    <Icon
                        v-if="loading"
                        name="svg-spinners:ring-resize"
                        class="text-text-hint size-4"
                    />
                </template>
            </DesignInput>

            <!-- `sm:contents` dissolves this row above the phone breakpoint so the
                 count returns to being a direct flex child, as it was before. -->
            <div class="flex items-center justify-between gap-3 sm:contents">
                <DesignButton
                    v-if="isPhone"
                    variant="secondary"
                    size="small"
                    icon="tabler:filter"
                    @click="filtersOpen = true"
                >
                    {{ t("vault.filters")
                    }}<span v-if="hasFilter">
                        ({{ selectedTypes.length }})</span
                    >
                </DesignButton>
                <p class="text-text-muted shrink-0 text-sm sm:text-base">
                    {{
                        t("vault.resultsRange", {
                            from: formatNumber(rangeFrom),
                            to: formatNumber(rangeTo),
                            total: formatNumber(totalHits),
                        })
                    }}
                </p>
            </div>
        </div>

        <div class="flex items-start">
            <!-- Filter sidebar — phones get the same component in a sheet below. -->
            <aside
                class="border-border-1 hidden w-64 shrink-0 self-stretch border-r p-6 md:block"
            >
                <VaultFilters
                    v-model:selected="selectedTypes"
                    :items="categoryItems"
                    :facets="facets"
                    :has-filter="hasFilter"
                    @clear="clearFilters"
                />
            </aside>

            <!-- Results -->
            <main class="min-w-0 flex-1 p-4 sm:p-6">
                <div v-if="loading" :class="gridClass">
                    <div
                        v-for="n in 15"
                        :key="n"
                        class="border-border-1 overflow-hidden rounded-[14px] border"
                    >
                        <DesignSkeleton
                            class="aspect-16/10 w-full rounded-none!"
                        />
                        <div class="flex flex-col gap-2 p-3">
                            <DesignSkeleton class="mb-1 h-4 w-3/4" />
                            <DesignSkeleton class="h-3 w-1/2" />
                            <DesignSkeleton class="h-3 w-1/2" />
                        </div>
                    </div>
                </div>

                <div v-else-if="items.length" :class="gridClass">
                    <VaultCard
                        v-for="item in items"
                        :key="item.VXID"
                        :item="item"
                        :base="base"
                    />
                </div>

                <div
                    v-else-if="loaded"
                    class="text-text-muted py-20 text-center"
                >
                    <Icon name="tabler:search" class="size-10 opacity-50" />
                    <p class="mt-3 text-sm">{{ t("vault.noResults") }}</p>
                </div>

                <!-- Pagination -->
                <div v-if="totalPages > 1" class="mt-6 flex justify-center">
                    <DesignPagination
                        v-model:page="page"
                        :total="totalHits"
                        :page-size="pageSize"
                        :sibling-count="1"
                        show-edges
                        :disabled="loading"
                    />
                </div>
            </main>
        </div>

        <!-- Filters apply live through the existing `watch(selectedTypes)`, so
             this button only dismisses — there is nothing to "apply". -->
        <DesignSheet v-if="isPhone" v-model:open="filtersOpen">
            <VaultFilters
                v-model:selected="selectedTypes"
                :items="categoryItems"
                :facets="facets"
                :has-filter="hasFilter"
                @clear="clearFilters"
            />
            <DesignButton class="mt-4 w-full" @click="filtersOpen = false">
                {{ t("vault.done") }}
            </DesignButton>
        </DesignSheet>
    </div>
</template>
