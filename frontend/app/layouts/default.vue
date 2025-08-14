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
    <div class="flex grow-1 flex-col">
        <header
            ref="header"
            class="border-accented bg-default sticky top-0 z-10 flex items-center gap-4 border-b px-4"
        >
            <NuxtLink to="/" class="flex items-center gap-2 text-sm font-bold">
                <img
                    src="/images/logo.png"
                    width="24"
                    height="24"
                    class="rounded-full"
                />
                <p>BCC Media Tools</p>
            </NuxtLink>
            <UNavigationMenu
                orientation="horizontal"
                highlight
                :items="enabledTools"
            />
            <ThemeSwitch class="ml-auto" />
            <LanguageSwitcher />
        </header>
        <main class="grow-1">
            <slot />
        </main>
    </div>
</template>
