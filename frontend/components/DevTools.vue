<script setup lang="ts">
import { BccButton } from "@bcc-code/design-library-vue";

const { $toast } = useNuxtApp();

const show = ref(false);

const clearLocalStorage = () => {
    localStorage.clear();
    $toast.success("Local storage cleared");
};
</script>

<template>
    <div class="fixed bottom-8 left-8 flex flex-col items-start gap-2">
        <transition
            enter-active-class="transition duration-200 ease-out origin-bottom-left"
            leave-active-class="transition duration-200 ease-out origin-bottom-left"
            enter-from-class="opacity-0 scale-95 translate-y-2"
            leave-to-class="opacity-0 scale-95 translate-y-2"
        >
            <div
                v-if="show"
                class="flex min-w-64 flex-col gap-2 rounded-lg border border-gray-300 bg-white p-4 shadow-lg"
            >
                <h4 class="font-bold">Developer tools</h4>
                <BccButton
                    size="sm"
                    variant="secondary"
                    @click="clearLocalStorage"
                >
                    Clear local storage
                </BccButton>
            </div>
        </transition>
        <BccButton size="lg" variant="secondary" rounded @click="show = !show">
            <Icon name="tabler:tool" />
        </BccButton>
    </div>
</template>
