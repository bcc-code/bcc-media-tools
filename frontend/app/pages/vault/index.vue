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

const MEDIA_CATEGORIES = ["video", "audio", "image"] as const;
type MediaCategory = (typeof MEDIA_CATEGORIES)[number];

const query = useQueryRef("q", "");
const selectedTypes = useQueryRef<MediaCategory[]>("types", []);
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

function facetCount(type: string): number {
    return facets.value.find((f) => f.mediaType === type)?.count ?? 0;
}

const hasFilter = computed(() => selectedTypes.value.length > 0);
function clearFilters() {
    selectedTypes.value = [];
}

function toggleType(value: MediaCategory, checked: boolean) {
    selectedTypes.value = checked
        ? [...selectedTypes.value, value]
        : selectedTypes.value.filter((v) => v !== value);
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
        <div class="border-border-1 border-b px-6 py-5">
            <DesignInput
                v-model="query"
                leading-icon="tabler:search"
                :placeholder="t('vault.searchPlaceholder')"
                class="w-full max-w-xl"
            >
                <template #trailing>
                    <Icon
                        v-if="loading"
                        name="svg-spinners:ring-resize"
                        class="text-text-hint size-4"
                    />
                </template>
            </DesignInput>
        </div>

        <div class="flex items-start">
            <!-- Filter sidebar -->
            <aside
                class="border-border-1 w-64 shrink-0 self-stretch border-r p-6"
            >
                <div class="mb-5 flex items-center justify-between">
                    <h2 class="text-title-1 text-text-default font-semibold">
                        {{ t("vault.filters") }}
                    </h2>
                    <button
                        v-if="hasFilter"
                        class="text-text-muted hover:text-text-default text-xs"
                        @click="clearFilters"
                    >
                        {{ t("vault.clear") }}
                    </button>
                </div>
                <fieldset>
                    <legend
                        class="text-text-muted mb-3 text-[11px] font-semibold tracking-wide uppercase"
                    >
                        {{ t("vault.mediaType") }}
                    </legend>
                    <div class="flex flex-col gap-y-2">
                        <div
                            v-for="item in categoryItems"
                            :key="item.value"
                            class="flex w-full items-center justify-between gap-2"
                        >
                            <DesignCheckbox
                                :model-value="
                                    selectedTypes.includes(item.value)
                                "
                                :label="item.label"
                                @update:model-value="
                                    toggleType(item.value, $event)
                                "
                            />
                            <span class="text-text-muted font-mono text-xs">
                                {{ facetCount(item.value) }}
                            </span>
                        </div>
                    </div>
                </fieldset>
            </aside>

            <!-- Results -->
            <main class="min-w-0 flex-1 p-6">
                <div class="mb-4 flex items-baseline justify-between">
                    <h2 class="text-text-default text-sm font-semibold">
                        {{ t("vault.results") }}
                    </h2>
                    <span class="text-text-muted">
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
                <div
                    v-if="loading"
                    class="grid grid-cols-[repeat(auto-fit,minmax(400px,1fr))] gap-4"
                >
                    <div
                        v-for="n in 15"
                        :key="n"
                        class="border-border-1 overflow-hidden rounded-[14px] border"
                    >
                        <USkeleton class="aspect-16/10 w-full rounded-none" />
                        <div class="flex flex-col gap-2 p-3">
                            <USkeleton class="mb-1 h-4 w-3/4" />
                            <USkeleton class="h-3 w-1/2" />
                            <USkeleton class="h-3 w-1/2" />
                        </div>
                    </div>
                </div>

                <div
                    v-else-if="items.length"
                    class="grid grid-cols-[repeat(auto-fit,minmax(400px,1fr))] gap-4"
                >
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
    </div>
</template>
