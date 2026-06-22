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
}

const props = withDefaults(defineProps<Props>(), {
    placeholder: "Velg...",
    placement: "bottom-start",
});

// Single-value model (mirrors Nuxt UI's USelect ergonomics). Ark works with
// arrays internally, so we adapt at the boundary.
const model = defineModel<string>({ default: "" });

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

const arrayModel = computed<string[]>({
    get: () => (model.value ? [model.value] : []),
    set: (v) => {
        model.value = v[0] ?? "";
    },
});

const selectedItem = computed<SelectItem | undefined>(() =>
    normalized.value.find((item) => item.value === model.value),
);
</script>

<template>
    <Select.Root
        v-model="arrayModel"
        :collection="collection"
        :disabled="disabled"
        :positioning="{ placement, gutter: 4 }"
    >
        <Select.Control>
            <Select.Trigger
                class="gradient-border bg-surface-raise text-title-3 text-text-default shadow-resting ds-focus-ring inline-flex w-full cursor-pointer items-center justify-between gap-2 rounded-xl px-3 py-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
                {{ selectedItem?.label ?? placeholder }}
                <Icon
                    name="tabler:chevron-down"
                    class="text-text-muted size-4"
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
                            v-for="item in normalized"
                            :key="item.value"
                            :item="item"
                            class="text-body-3 text-text-default data-highlighted:bg-surface-indent flex cursor-pointer items-center justify-between gap-2 rounded-lg px-3 py-2"
                        >
                            <Select.ItemText>{{ item.label }}</Select.ItemText>
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
