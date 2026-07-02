<script setup lang="ts">
import { Combobox, createListCollection } from "@ark-ui/vue";
import { useDebounceFn } from "@vueuse/core";
import type { Placement } from "~/types/design";

export interface ComboboxItem {
    // Canonical value stored on selection (e.g. an entity id).
    value: string;
    // Display text shown in the input and as the primary line.
    label: string;
    // Optional secondary line (e.g. the canonical reference).
    detail?: string;
}

interface Props {
    // The visible text — parent-controlled (the marker label). Free-form, so
    // the field doubles as a plain input when nothing is linked.
    modelValue: string;
    // Suggestions to show; the parent refreshes these in response to `search`.
    items: ComboboxItem[];
    fieldLabel?: string;
    placeholder?: string;
    placement?: Placement;
    disabled?: boolean;
    loading?: boolean;
    leadingIcon?: string;
    // Debounce for the `search` emit, in ms.
    debounce?: number;
    // Message shown when a query returned no suggestions.
    emptyText?: string;
}

const props = withDefaults(defineProps<Props>(), {
    fieldLabel: undefined,
    placeholder: undefined,
    placement: "bottom-start",
    debounce: 300,
    leadingIcon: undefined,
    emptyText: undefined,
});

const emit = defineEmits<{
    // Fired on genuine user typing (not on programmatic selection sync). Update
    // the bound value and refresh suggestions in response.
    "update:modelValue": [value: string];
    // Debounced, fired as the user types. The parent should fetch suggestions.
    search: [query: string];
    // Fired when a suggestion is picked.
    select: [item: ComboboxItem];
}>();

const collection = computed(() =>
    createListCollection({
        items: props.items,
        itemToValue: (item) => item.value,
        itemToString: (item) => item.label,
    }),
);

const emitSearch = useDebounceFn(
    (q: string) => emit("search", q),
    () => Math.max(0, props.debounce),
);

// Picking an item makes Ark sync the input text, which fires an input-value
// change we must NOT treat as user typing. This flag swallows that echo
// regardless of whether value-change or input-value-change fires first.
let selecting = false;

function onInputValueChange(details: { inputValue: string }) {
    if (selecting) return; // echo from a selection — `selecting` clears on nextTick
    emit("update:modelValue", details.inputValue);
    emitSearch(details.inputValue);
}

function onValueChange(details: { value: string[] }) {
    const picked = props.items.find((i) => i.value === details.value[0]);
    if (!picked) return;
    // Guard the input-value echo Ark emits on selection, whichever order it
    // arrives in, then release the guard once this tick settles.
    selecting = true;
    nextTick(() => {
        selecting = false;
    });
    emit("select", picked);
}

const showEmpty = computed(
    () =>
        !props.loading &&
        !!props.emptyText &&
        !!props.modelValue.trim() &&
        props.items.length === 0,
);
</script>

<template>
    <Combobox.Root
        :collection="collection"
        :input-value="modelValue"
        :disabled="disabled"
        :placeholder="placeholder"
        :open-on-click="false"
        allow-custom-value
        select-on-blur
        :positioning="{ placement, gutter: 4 }"
        @input-value-change="onInputValueChange"
        @value-change="onValueChange"
    >
        <Combobox.Label
            v-if="fieldLabel"
            class="text-body-3 text-text-muted mb-1 block"
        >
            {{ fieldLabel }}
        </Combobox.Label>

        <Combobox.Control class="relative flex items-center">
            <Icon
                v-if="leadingIcon"
                :name="leadingIcon"
                class="text-text-hint pointer-events-none absolute left-3 size-4"
            />
            <Combobox.Input
                :placeholder="placeholder"
                :class="[
                    'border-border-1 text-body-3 text-text-default placeholder:text-text-hint ds-focus-ring w-full rounded-xl border py-2 disabled:cursor-not-allowed disabled:opacity-50',
                    leadingIcon ? 'pl-9' : 'pl-3',
                    'pr-9',
                ]"
            />
            <div class="absolute right-2 flex items-center gap-1">
                <Icon
                    v-if="loading"
                    name="tabler:loader-2"
                    class="text-text-hint size-4 animate-spin"
                />
                <slot name="trailing" />
            </div>
        </Combobox.Control>

        <Teleport to="#teleports">
            <Combobox.Positioner>
                <Combobox.Content
                    class="gradient-border bg-surface-raise shadow-floating ease-out-expo z-50 max-h-72 min-w-[var(--reference-width)] origin-[--transform-origin] overflow-y-auto rounded-xl p-1 transition-[opacity,transform] duration-200 data-[state=closed]:scale-95 data-[state=closed]:opacity-0 data-[state=open]:scale-100 data-[state=open]:opacity-100"
                >
                    <Combobox.Item
                        v-for="item in items"
                        :key="item.value"
                        :item="item"
                        class="text-body-3 text-text-default data-highlighted:bg-surface-indent flex cursor-pointer items-center justify-between gap-3 rounded-lg px-3 py-2"
                    >
                        <span class="min-w-0 flex-1 truncate">
                            <Combobox.ItemText>
                                {{ item.label }}
                            </Combobox.ItemText>
                        </span>
                        <span
                            v-if="item.detail"
                            class="text-text-hint text-caption-1 shrink-0 tabular-nums"
                        >
                            {{ item.detail }}
                        </span>
                    </Combobox.Item>

                    <div
                        v-if="showEmpty"
                        class="text-text-hint text-body-3 px-3 py-2"
                    >
                        {{ emptyText }}
                    </div>
                </Combobox.Content>
            </Combobox.Positioner>
        </Teleport>
    </Combobox.Root>
</template>
