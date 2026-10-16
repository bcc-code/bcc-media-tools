<script setup lang="ts">
import { Dialog } from "@ark-ui/vue";

// A centred modal on desktop that becomes a draggable bottom sheet below `md:`
// — same props, same slot, so callers need no branching of their own.
interface Props {
    title?: string;
    description?: string;
    size?: "sm" | "md" | "lg" | "xl";
    /** Shown in a tinted circle above the title. */
    icon?: string;
    iconVariant?: "success" | "warning" | "info" | "error" | "neutral";
    align?: "start" | "center";
}

const props = withDefaults(defineProps<Props>(), {
    title: undefined,
    description: undefined,
    size: "md",
    icon: undefined,
    iconVariant: "neutral",
    align: "start",
});

// Every size collapses to the same width on a phone, so below `md:` the sheet
// form takes over entirely rather than rendering a cramped centred card.
const isPhone = useIsPhone();

const maxWidthClass = computed(
    () =>
        ({
            sm: "max-w-sm",
            md: "max-w-lg",
            lg: "max-w-2xl",
            xl: "max-w-4xl",
        })[props.size],
);

const iconClass = computed(
    () =>
        ({
            success: "bg-semantic-success/15 text-semantic-success",
            warning: "bg-semantic-warning/15 text-semantic-warning",
            info: "bg-semantic-info/15 text-semantic-info",
            error: "bg-semantic-error/15 text-semantic-error",
            neutral: "bg-surface-indent text-text-muted",
        })[props.iconVariant],
);

const open = defineModel<boolean>("open", { default: false });

const initialFocusTarget = ref<HTMLElement | null>(null);

const setInitialFocus = (el: unknown) => {
    if (el == null) {
        initialFocusTarget.value = null;
        return;
    }
    if (el instanceof HTMLElement) {
        initialFocusTarget.value = el;
        return;
    }
    const $el = (el as { $el?: unknown }).$el;
    if ($el instanceof HTMLElement) {
        initialFocusTarget.value = $el;
    }
};

const initialFocusEl = () => initialFocusTarget.value;

watch(open, async (isOpen) => {
    if (!isOpen) return;
    await nextTick();
    requestAnimationFrame(() => {
        initialFocusTarget.value?.focus();
    });
});
</script>

<template>
    <DesignSheet
        v-if="isPhone"
        v-model:open="open"
        :title="title"
        :description="description"
        :icon="icon"
        :icon-variant="iconVariant"
        :align="align"
    >
        <template #trigger><slot name="trigger" /></template>
        <template #default="{ initialFocus }">
            <slot :initial-focus="initialFocus" />
        </template>
    </DesignSheet>

    <Dialog.Root
        v-else
        v-model:open="open"
        lazy-mount
        unmount-on-exit
        :initial-focus-el="initialFocusEl"
    >
        <slot name="trigger" />

        <Teleport to="#teleports">
            <Dialog.Backdrop
                class="data-[state=closed]:fade-exit data-[state=open]:fade-enter fixed inset-0 z-40 bg-black/40"
            />
            <Dialog.Positioner
                class="fixed inset-0 z-50 flex items-center justify-center p-4"
            >
                <!-- Keyframes, not transitions: Ark's presence layer unmounts
                     on close unless it sees a running CSS animation. -->
                <Dialog.Content
                    class="gradient-border bg-surface-raise shadow-floating data-[state=closed]:dialog-exit data-[state=open]:dialog-enter relative flex max-h-[85dvh] w-full origin-center flex-col overflow-hidden rounded-2xl"
                    :class="maxWidthClass"
                >
                    <div class="overflow-y-auto overscroll-contain p-6">
                        <div
                            v-if="title || description || icon"
                            class="mb-5"
                            :class="
                                align === 'center'
                                    ? 'flex flex-col items-center text-center'
                                    : ''
                            "
                        >
                            <div
                                v-if="icon"
                                class="mb-4 flex size-14 items-center justify-center rounded-full"
                                :class="iconClass"
                            >
                                <Icon :name="icon" class="text-3xl" />
                            </div>
                            <Dialog.Title
                                v-if="title"
                                class="text-heading-3 text-text-default"
                                :class="
                                    align === 'center'
                                        ? 'text-balance'
                                        : 'pr-10'
                                "
                            >
                                {{ title }}
                            </Dialog.Title>
                            <Dialog.Description
                                v-if="description"
                                class="text-body-3 text-text-muted mt-1"
                                :class="
                                    align === 'center'
                                        ? 'mx-auto max-w-[36ch] text-pretty'
                                        : ''
                                "
                            >
                                {{ description }}
                            </Dialog.Description>
                        </div>

                        <slot :initial-focus="setInitialFocus" />
                    </div>

                    <!-- Anchored by `relative` on Content: without it this
                         resolves against the full-screen positioner and lands in
                         the viewport corner instead of the card's. -->
                    <Dialog.CloseTrigger
                        class="text-text-hint hover:text-text-default ds-focus-ring absolute top-4 right-4 cursor-pointer rounded-lg p-1"
                    >
                        <Icon name="tabler:x" class="size-5" />
                    </Dialog.CloseTrigger>
                </Dialog.Content>
            </Dialog.Positioner>
        </Teleport>
    </Dialog.Root>
</template>
