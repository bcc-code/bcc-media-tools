<script setup>
const colorMode = useColorMode();

const isDark = computed({
    get() {
        return colorMode.value === "dark";
    },
    set(_isDark) {
        colorMode.preference = _isDark ? "dark" : "light";
    },
});
</script>

<template>
    <ClientOnly v-if="!colorMode?.forced">
        <DesignButton
            :icon="isDark ? 'tabler:moon' : 'tabler:sun'"
            variant="tertiary"
            @click="isDark = !isDark"
        />

        <template #fallback>
            <!-- Matches the real tertiary medium DesignButton box (px-4 py-2.5 +
                 a 1em icon). A smaller placeholder makes --header-height jump on
                 hydration, reflowing every `100dvh - var(--header-height)` page. -->
            <div class="h-10 w-12" />
        </template>
    </ClientOnly>
</template>
