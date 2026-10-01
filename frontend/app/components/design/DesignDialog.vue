<script setup lang="ts">
import { Dialog } from "@ark-ui/vue";

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
    <Dialog.Root
        v-model:open="open"
        lazy-mount
        unmount-on-exit
        :initial-focus-el="initialFocusEl"
    >
        <slot name="trigger" />

        <Teleport to="#teleports">
            <Dialog.Backdrop
                class="fixed inset-0 z-40 bg-black/40 transition-opacity duration-200 data-[state=closed]:opacity-0 data-[state=open]:opacity-100"
            />
            <Dialog.Positioner
                class="fixed inset-0 z-50 flex items-center justify-center p-4"
            >
                <Dialog.Content
                    class="gradient-border bg-surface-raise shadow-floating ease-out-expo max-h-[85vh] w-full origin-center overflow-y-auto rounded-2xl p-6 transition-[opacity,transform] duration-200 data-[state=closed]:scale-95 data-[state=closed]:opacity-0 data-[state=open]:scale-100 data-[state=open]:opacity-100"
                    :class="maxWidthClass"
                >
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
                            :class="align === 'center' ? 'text-balance' : ''"
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
