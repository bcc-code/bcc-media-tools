<script setup lang="ts">
import { Menu } from "@ark-ui/vue";
import type { Placement } from "~/types/design";

export interface MenuItem {
    value: string;
    label: string;
    icon?: string;
    intent?: "neutral" | "danger";
    disabled?: boolean;
}

interface Props {
    items: MenuItem[];
    placement?: Placement;
    // Default trigger icon; overridden by the #trigger slot.
    triggerIcon?: string;
    triggerLabel?: string;
}

withDefaults(defineProps<Props>(), {
    placement: "bottom-end",
    triggerIcon: "tabler:dots-vertical",
    triggerLabel: undefined,
});

const emit = defineEmits<{ select: [value: string] }>();
</script>

<template>
    <Menu.Root
        :positioning="{ placement, gutter: 4 }"
        @select="(d) => emit('select', d.value)"
    >
        <Menu.Trigger
            :aria-label="triggerLabel"
            class="text-text-default hover:bg-surface-indent ds-focus-ring ease-out-expo inline-flex cursor-pointer items-center justify-center rounded-3xl px-4 py-2.5 transition-transform duration-200 active:scale-95"
        >
            <slot name="trigger">
                <Icon :name="triggerIcon" class="size-5 shrink-0" />
            </slot>
        </Menu.Trigger>

        <Teleport to="#teleports">
            <Menu.Positioner>
                <Menu.Content
                    class="gradient-border bg-surface-raise shadow-floating ease-out-expo z-50 min-w-44 origin-[--transform-origin] rounded-xl p-1 transition-[opacity,transform] duration-200 focus:outline-none data-[state=closed]:scale-95 data-[state=closed]:opacity-0 data-[state=open]:scale-100 data-[state=open]:opacity-100"
                >
                    <Menu.Item
                        v-for="item in items"
                        :key="item.value"
                        :value="item.value"
                        :disabled="item.disabled"
                        class="text-body-3 data-highlighted:bg-surface-indent flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2 data-disabled:cursor-not-allowed data-disabled:opacity-50"
                        :class="
                            item.intent === 'danger'
                                ? 'text-semantic-error'
                                : 'text-text-default'
                        "
                    >
                        <Icon
                            v-if="item.icon"
                            :name="item.icon"
                            class="size-4 shrink-0"
                        />
                        {{ item.label }}
                    </Menu.Item>
                </Menu.Content>
            </Menu.Positioner>
        </Teleport>
    </Menu.Root>
</template>
