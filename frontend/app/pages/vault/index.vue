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

const query = ref("");
const selectedTypes = ref<string[]>([]);
const page = ref(1);

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
}, { deep: true });
watch(page, load);
onMounted(load);

function facetCount(type: string): number {
    return facets.value.find((f) => f.mediaType === type)?.count ?? 0;
}

function toggleType(type: string) {
    const i = selectedTypes.value.indexOf(type);
    if (i === -1) selectedTypes.value = [...selectedTypes.value, type];
    else selectedTypes.value = selectedTypes.value.filter((x) => x !== type);
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
                icon="i-lucide-search"
                size="lg"
                :loading="loading"
                :placeholder="t('vault.searchPlaceholder')"
                class="w-full max-w-xl"
            />
        </div>

        <div class="flex items-start">
            <!-- Filter sidebar -->
            <aside class="border-default w-64 shrink-0 self-stretch border-r p-6">
                <div class="mb-5 flex items-center justify-between">
                    <h2 class="text-lg font-semibold">{{ t("vault.filters") }}</h2>
                    <button
                        v-if="hasFilter"
                        class="text-muted hover:text-default text-xs"
                        @click="clearFilters"
                    >
                        {{ t("vault.clear") }}
                    </button>
                </div>
                <h3
                    class="text-muted mb-3 text-[11px] font-semibold uppercase tracking-wide"
                >
                    {{ t("vault.mediaType") }}
                </h3>
                <div class="flex flex-col gap-0.5">
                    <button
                        v-for="cat in MEDIA_CATEGORIES"
                        :key="cat"
                        class="hover:bg-muted -mx-2 flex h-[34px] items-center gap-2.5 rounded-md px-2 text-sm"
                        @click="toggleType(cat)"
                    >
                        <span
                            class="flex size-[18px] shrink-0 items-center justify-center rounded"
                            :class="
                                selectedTypes.includes(cat)
                                    ? 'bg-primary'
                                    : 'border-accented border'
                            "
                        >
                            <UIcon
                                v-if="selectedTypes.includes(cat)"
                                name="i-lucide-check"
                                class="text-inverted size-3"
                            />
                        </span>
                        <span class="flex-1 text-left">{{
                            t(`vault.type.${cat}`)
                        }}</span>
                        <span class="text-muted font-mono text-xs">{{
                            facetCount(cat)
                        }}</span>
                    </button>
                </div>
            </aside>

            <!-- Results -->
            <main class="min-w-0 flex-1 p-6">
                <div class="mb-4 flex items-baseline justify-between">
                    <h2 class="text-sm font-semibold">{{ t("vault.results") }}</h2>
                    <span class="text-muted text-xs">
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
                    class="grid grid-cols-5 gap-4"
                >
                    <div
                        v-for="n in 10"
                        :key="n"
                        class="border-default overflow-hidden rounded-[14px] border"
                    >
                        <USkeleton class="aspect-[16/10] w-full rounded-none" />
                        <div class="space-y-2 p-3">
                            <USkeleton class="h-3 w-3/4" />
                            <USkeleton class="h-2 w-1/2" />
                        </div>
                    </div>
                </div>

                <div
                    v-else-if="items.length"
                    class="grid grid-cols-5 gap-4"
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
                    class="text-muted py-20 text-center"
                >
                    <UIcon
                        name="i-lucide-search"
                        class="size-10 opacity-50"
                    />
                    <p class="mt-3 text-sm">{{ t("vault.noResults") }}</p>
                </div>

                <!-- Pagination -->
                <div
                    v-if="totalPages > 1"
                    class="mt-6 flex items-center justify-center gap-3"
                >
                    <UButton
                        icon="i-lucide-chevron-left"
                        color="neutral"
                        variant="outline"
                        :disabled="page <= 1 || loading"
                        @click="page--"
                    >
                        {{ t("vault.previous") }}
                    </UButton>
                    <span class="text-muted font-mono text-sm">
                        {{ page }} / {{ totalPages }}
                    </span>
                    <UButton
                        trailing-icon="i-lucide-chevron-right"
                        color="neutral"
                        variant="outline"
                        :disabled="page >= totalPages || loading"
                        @click="page++"
                    >
                        {{ t("vault.next") }}
                    </UButton>
                </div>
            </main>
        </div>
    </div>
</template>
