<script lang="ts" setup>
// The vault facet filters, rendered in two places: a fixed sidebar from `md:`
// up, and inside a DesignSheet on phones. The host supplies the padding.
import type { VaultFacet } from "~~/src/gen/api/v1/api_pb";

const props = defineProps<{
    items: { value: string; label: string }[];
    facets: VaultFacet[];
    hasFilter: boolean;
}>();

defineEmits<{ clear: [] }>();

const selected = defineModel<string[]>("selected", { required: true });

const { t } = useI18n();
const { formatNumber } = useNumberFormat();

function facetCount(type: string): number {
    return props.facets.find((f) => f.mediaType === type)?.count ?? 0;
}

function toggleType(value: string, checked: boolean) {
    selected.value = checked
        ? [...selected.value, value]
        : selected.value.filter((v) => v !== value);
}
</script>

<template>
    <div>
        <div class="mb-5 flex items-center justify-between gap-2">
            <h2 class="text-title-1 text-text-default font-semibold">
                {{ t("vault.filters") }}
            </h2>
            <button
                v-if="hasFilter"
                class="text-text-muted hover:text-text-default text-xs"
                @click="$emit('clear')"
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
                    v-for="item in items"
                    :key="item.value"
                    class="flex w-full items-center justify-between gap-2"
                >
                    <DesignCheckbox
                        :model-value="selected.includes(item.value)"
                        :label="item.label"
                        @update:model-value="toggleType(item.value, $event)"
                    />
                    <span class="text-text-muted font-mono text-xs">
                        {{ formatNumber(facetCount(item.value)) }}
                    </span>
                </div>
            </div>
        </fieldset>
    </div>
</template>
