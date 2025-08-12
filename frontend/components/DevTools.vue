<script setup lang="ts">
const show = ref(false);

const toast = useToast();
const clearLocalStorage = () => {
    localStorage.clear();
    toast.add({
        icon: "heroicons:check",
        title: "Cleared local storage",
        description:
            "All relevant data has been cleared from the local storage",
        color: "success",
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
                class="border-default bg-default flex min-w-64 flex-col gap-2 rounded-lg border p-4 shadow-lg"
            >
                <h4 class="font-bold">Developer tools</h4>
                <UButton
                    size="sm"
                    variant="soft"
                    block
                    @click="clearLocalStorage"
                >
                    Clear local storage
                </UButton>
            </div>
        </transition>
        <UButton size="xl" variant="solid" square @click="show = !show">
            <Icon name="tabler:tool" />
        </UButton>
    </div>
</template>
