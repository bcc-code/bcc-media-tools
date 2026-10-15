<script setup lang="ts">
import { Dialog } from "@ark-ui/vue";

// A dialog that slides up from the bottom of the screen: the phone
// counterpart of DesignDialog, for secondary actions within thumb reach.
interface Props {
    title?: string;
    description?: string;
}

withDefaults(defineProps<Props>(), {
    title: undefined,
    description: undefined,
});

const open = defineModel<boolean>("open", { default: false });
</script>

<template>
    <Dialog.Root v-model:open="open" lazy-mount unmount-on-exit>
        <Teleport to="#teleports">
            <Dialog.Backdrop
                class="fixed inset-0 z-40 bg-black/40 transition-opacity duration-200 data-[state=closed]:opacity-0 data-[state=open]:opacity-100"
            />
            <Dialog.Positioner
                class="fixed inset-x-0 bottom-0 z-50 flex justify-center"
            >
                <Dialog.Content
                    class="gradient-border bg-surface-raise shadow-floating ease-out-expo max-h-[85dvh] w-full max-w-lg overflow-y-auto rounded-t-3xl px-4 pt-2 pb-[max(1.5rem,env(safe-area-inset-bottom))] transition-transform duration-300 data-[state=closed]:translate-y-full data-[state=open]:translate-y-0"
                >
                    <div
                        class="bg-border-1 mx-auto mb-3 h-1.5 w-10 rounded-full"
                        aria-hidden="true"
                    />
                    <div v-if="title || description" class="mb-4">
                        <Dialog.Title
                            v-if="title"
                            class="text-title-1 text-text-default"
                        >
                            {{ title }}
                        </Dialog.Title>
                        <Dialog.Description
                            v-if="description"
                            class="text-caption-1 text-text-hint mt-1"
                        >
                            {{ description }}
                        </Dialog.Description>
                    </div>

                    <slot />
                </Dialog.Content>
            </Dialog.Positioner>
        </Teleport>
    </Dialog.Root>
</template>
