<script setup lang="ts">
// Set header height CSS variable
const header = useTemplateRef("header");
const { height } = useElementSize(header);
watch(
    height,
    (h) => {
        document.documentElement.style.setProperty("--header-height", `${h}px`);
    },
    { immediate: true },
);

const { enabledTools } = useTools();
</script>

<template>
    <div class="flex grow flex-col">
        <header
            ref="header"
            class="border-accented bg-default sticky top-0 z-10 flex items-center gap-4 border-b px-4"
        >
            <NuxtLink to="/" class="flex items-center">
                <AppLogo class="h-5 w-max" />
            </NuxtLink>
            <UNavigationMenu
                orientation="horizontal"
                highlight
                :items="enabledTools"
            />
            <ThemeSwitch class="ml-auto" />
            <LanguageSwitcher />
        </header>
        <main class="grow">
            <slot />
        </main>
    </div>
</template>
