<script setup lang="ts">
import { Select, createListCollection } from "@ark-ui/vue";
import type { Placement } from "~/types/design";

export interface SelectItem {
    label: string;
    value: string;
}

interface Props {
    // Accepts plain strings (label === value) or {label, value} objects.
    items: (string | SelectItem)[];
    placeholder?: string;
    placement?: Placement;
    disabled?: boolean;
    // Multi-select: model is a string[] and the trigger shows the selected
    // labels comma-separated (truncated with an ellipsis on overflow).
    multiple?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
    placeholder: "Velg...",
    placement: "bottom-start",
});

// Single-select binds a string; multi-select binds a string[].
const model = defineModel<string | string[]>({ default: "" });

const normalized = computed<SelectItem[]>(() =>
    props.items.map((i) =>
        typeof i === "string" ? { label: i, value: i } : i,
    ),
);

const collection = computed(() =>
    createListCollection({
        items: normalized.value,
        itemToString: (item) => item.label,
        itemToValue: (item) => item.value,
    }),
);

// Ark works with arrays internally; adapt to the single/multi model at the edge.
const arrayModel = computed<string[]>({
    get: () => {
        if (props.multiple)
            return Array.isArray(model.value) ? model.value : [];
        return model.value ? [model.value as string] : [];
    },
    set: (v) => {
        if (props.multiple) model.value = v;
        else model.value = v[0] ?? "";
    },
});

const selectedItems = computed<SelectItem[]>(() =>
    normalized.value.filter((item) => arrayModel.value.includes(item.value)),
);

const hasSelection = computed(() => selectedItems.value.length > 0);

const displayText = computed(() => {
    if (!hasSelection.value) return props.placeholder;
    return selectedItems.value.map((i) => i.label).join(", ");
});
</script>

<template>
    <Select.Root
        v-model="arrayModel"
        :collection="collection"
        :multiple="multiple"
        :disabled="disabled"
        :positioning="{ placement, gutter: 4 }"
    >
        <Select.Control>
            <Select.Trigger
                class="gradient-border bg-surface-raise text-title-3 shadow-resting ds-focus-ring inline-flex w-full cursor-pointer items-center justify-between gap-2 rounded-xl px-3 py-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
                <span
                    class="min-w-0 flex-1 truncate text-left"
                    :class="
                        hasSelection ? 'text-text-default' : 'text-text-hint'
                    "
                >
                    {{ displayText }}
                </span>
                <Icon
                    name="tabler:chevron-down"
                    class="text-text-muted size-4 shrink-0"
                />
            </Select.Trigger>
        </Select.Control>

        <Teleport to="#teleports">
            <Select.Positioner>
                <Select.Content
                    class="gradient-border bg-surface-raise shadow-floating ease-out-expo z-50 min-w-[var(--reference-width)] origin-[--transform-origin] rounded-xl p-1 transition-[opacity,transform] duration-200 data-[state=closed]:scale-95 data-[state=closed]:opacity-0 data-[state=open]:scale-100 data-[state=open]:opacity-100"
                >
                    <Select.ItemGroup>
                        <Select.Item
                            v-for="(item, i) in normalized"
                            :key="item.value"
                            :item="item"
                            class="text-body-3 text-text-default data-highlighted:bg-surface-indent flex cursor-pointer items-center justify-between gap-2 rounded-lg px-3 py-2"
                        >
                            <slot name="item" :item="items[i]" :normalized="item">
                                <Select.ItemText>
                                    {{ item.label }}
                                </Select.ItemText>
                            </slot>
                            <Select.ItemIndicator>
                                <Icon
                                    name="tabler:check"
                                    class="text-primary-contrast size-4"
                                />
                            </Select.ItemIndicator>
                        </Select.Item>
                    </Select.ItemGroup>
                </Select.Content>
            </Select.Positioner>
        </Teleport>
    </Select.Root>
</template>
