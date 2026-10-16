<script setup lang="ts">
import { Drawer } from "@ark-ui/vue";

// A bottom sheet: the phone counterpart of DesignDialog, and what DesignDialog
// itself becomes below `md:`.
//
// Built on Ark's Drawer rather than Dialog, so swipe-to-dismiss, the rubber-band
// over-pull and the drag-vs-scroll arbitration come from the state machine. The
// machine owns the content's `transform` (it writes `--drawer-translate-*` as
// your finger moves); the enter/exit slide below is the independent `translate`
// property, which composes with it instead of fighting it.
interface Props {
    title?: string;
    description?: string;
    /** Shown in a tinted circle above the title. */
    icon?: string;
    iconVariant?: "success" | "warning" | "info" | "error" | "neutral";
    align?: "start" | "center";
}

const props = withDefaults(defineProps<Props>(), {
    title: undefined,
    description: undefined,
    icon: undefined,
    iconVariant: "neutral",
    align: "start",
});

const open = defineModel<boolean>("open", { default: false });

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
</script>

<template>
    <Drawer.Root
        :open="open"
        swipe-direction="down"
        lazy-mount
        unmount-on-exit
        :initial-focus-el="initialFocusEl"
        @open-change="open = $event.open"
    >
        <slot name="trigger" />

        <Teleport to="#teleports">
            <!-- Opacity is the resting scrim value (animated on enter/exit)
                 times the live swipe progress, so dragging the sheet away
                 lightens the scrim under your finger. -->
            <Drawer.Backdrop
                class="data-[state=closed]:scrim-exit data-[state=open]:scrim-enter fixed inset-0 z-40 bg-black [--sheet-scrim-opacity:0.4]"
                style="
                    opacity: calc(
                        var(--sheet-scrim-opacity) *
                            (1 - var(--drawer-swipe-progress, 0))
                    );
                "
            />
            <Drawer.Positioner
                class="fixed inset-x-0 bottom-0 z-50 flex justify-center"
            >
                <Drawer.Content
                    class="gradient-border bg-surface-raise shadow-floating data-[state=closed]:sheet-exit data-[state=open]:sheet-enter flex max-h-[85dvh] w-full max-w-lg flex-col overflow-hidden rounded-t-3xl transition-[transform] duration-300 ease-[cubic-bezier(0.32,0.72,0,1)] will-change-transform"
                >
                    <Drawer.Grabber
                        class="flex-none cursor-grab py-3 active:cursor-grabbing"
                    >
                        <Drawer.GrabberIndicator
                            class="bg-border-1 mx-auto h-1.5 w-10 rounded-full"
                        />
                    </Drawer.Grabber>

                    <div
                        v-if="title || description || icon"
                        class="flex-none px-4 pb-4"
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
                        <Drawer.Title
                            v-if="title"
                            class="text-title-1 text-text-default"
                            :class="align === 'center' ? 'text-balance' : ''"
                        >
                            {{ title }}
                        </Drawer.Title>
                        <Drawer.Description
                            v-if="description"
                            class="text-caption-1 text-text-hint mt-1"
                            :class="
                                align === 'center'
                                    ? 'mx-auto max-w-[36ch] text-pretty'
                                    : ''
                            "
                        >
                            {{ description }}
                        </Drawer.Description>
                    </div>

                    <!-- `overscroll-contain` stops a flick at the top of this
                         list from rubber-banding the page behind the sheet. -->
                    <div
                        class="overflow-y-auto overscroll-contain px-4 pb-[max(1.5rem,env(safe-area-inset-bottom))]"
                    >
                        <slot :initial-focus="setInitialFocus" />
                    </div>
                </Drawer.Content>
            </Drawer.Positioner>
        </Teleport>
    </Drawer.Root>
</template>
