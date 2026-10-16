<script setup lang="ts">
import { SegmentGroup } from "@ark-ui/vue";

export interface SegmentItem {
    label: string;
    value: string;
    icon?: string;
}

interface Props {
    // Plain strings (label === value) or {label, value, icon?} objects.
    items: (string | SegmentItem)[];
    disabled?: boolean;
}

const props = defineProps<Props>();

const model = defineModel<string>({ default: "" });

const normalized = computed<SegmentItem[]>(() =>
    props.items.map((i) =>
        typeof i === "string" ? { label: i, value: i } : i,
    ),
);
</script>

<template>
    <!-- The indicator is absolutely positioned from measured CSS vars, so the
         items genuinely cannot wrap: scroll the group instead. The indicator
         lives inside the scroller, so its offsets stay correct. -->
    <SegmentGroup.Root
        v-model="model"
        class="bg-surface-indent scrollbar-hide relative isolate inline-flex max-w-full touch-pan-x items-center gap-0.5 overflow-x-auto rounded-xl p-1"
    >
        <SegmentGroup.Indicator
            class="gradient-border bg-surface-raise shadow-resting absolute z-0 rounded-lg"
            style="
                width: var(--width);
                height: var(--height);
                top: var(--top);
                left: var(--left);
                transition:
                    left 250ms var(--ease-out-expo),
                    width 200ms var(--ease-out-expo),
                    height 200ms var(--ease-out-expo),
                    top 200ms var(--ease-out-expo);
            "
        />
        <SegmentGroup.Item
            v-for="item in normalized"
            :key="item.value"
            :value="item.value"
            class="text-title-3 text-text-muted data-[state=checked]:text-text-default relative z-1 inline-flex shrink-0 cursor-pointer items-center gap-1.5 rounded-lg px-3 py-1.5 select-none"
        >
            <Icon
                v-if="item.icon"
                :name="item.icon"
                class="relative z-1 size-4"
            />
            <SegmentGroup.ItemText class="relative z-1">
                {{ item.label }}
            </SegmentGroup.ItemText>
            <SegmentGroup.ItemControl class="hidden" />
            <SegmentGroup.ItemHiddenInput />
        </SegmentGroup.Item>
    </SegmentGroup.Root>
</template>
