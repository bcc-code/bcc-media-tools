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

const route = useRoute();
function isActive(to: string) {
    const p = to.replace(/\/$/, "");
    return route.path === p || route.path.startsWith(p + "/");
}
</script>

<template>
    <div class="flex grow flex-col">
        <header
            ref="header"
            class="border-border-1 bg-surface-default sticky top-0 z-10 flex items-center gap-4 border-b px-4 py-2"
        >
            <NuxtLink to="/" class="flex items-center">
                <AppLogo class="h-5 w-max" />
            </NuxtLink>
            <nav class="flex items-center gap-1">
                <NuxtLink
                    v-for="tool in enabledTools"
                    :key="tool.to"
                    :to="tool.to"
                    :aria-current="isActive(tool.to) ? 'page' : undefined"
                    class="text-title-3 ds-focus-ring flex items-center gap-2 rounded-xl px-3 py-2"
                    :class="
                        isActive(tool.to)
                            ? 'gradient-border bg-surface-raise text-text-default shadow-resting'
                            : 'text-text-muted hover:text-text-default'
                    "
                >
                    <Icon :name="tool.icon" class="size-4" />
                    {{ tool.label }}
                </NuxtLink>
            </nav>
            <ThemeSwitch class="ml-auto" />
            <LanguageSwitcher />
        </header>
        <main class="grow">
            <slot />
        </main>
    </div>
</template>
