<script setup lang="ts">
const show = ref(false);

const toaster = useToast();
const clearLocalStorage = () => {
    localStorage.clear();
    toaster.create({
        title: "Cleared local storage",
        description:
            "All relevant data has been cleared from the local storage",
        type: "success",
    });
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
                class="gradient-border bg-surface-raise shadow-floating flex min-w-64 flex-col gap-2 rounded-xl p-4"
            >
                <h4 class="text-text-default font-bold">Developer tools</h4>
                <DesignButton
                    variant="secondary"
                    size="small"
                    class="w-full"
                    @click="clearLocalStorage"
                >
                    Clear local storage
                </DesignButton>
            </div>
        </transition>
        <DesignButton size="large" icon="tabler:tool" @click="show = !show" />
    </div>
</template>
