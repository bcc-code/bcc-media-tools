<script setup lang="ts">
// Set header height CSS variable. Must be the border box: pages size themselves
// as `100dvh - var(--header-height)`, and a content-box measurement leaves out
// the header's padding and border, making them overflow by that much.
const header = useTemplateRef("header");
const { height } = useElementSize(header, undefined, { box: "border-box" });
watch(
    height,
    (h) => {
        document.documentElement.style.setProperty("--header-height", `${h}px`);
    },
    { immediate: true },
);

const { enabledTools } = useTools();

const { t } = useI18n();
const route = useRoute();

// On phones the tool links don't fit in one row: they move into a menu.
const toolMenuItems = computed(() =>
    enabledTools.value.map((tool) => ({
        value: tool.to,
        label: tool.label,
        icon: tool.icon,
    })),
);
function isActive(to: string) {
    const p = to.replace(/\/$/, "");
    return route.path === p || route.path.startsWith(p + "/");
}
</script>

<template>
    <div class="flex grow flex-col">
        <header
            ref="header"
            class="border-border-1 bg-surface-default sticky top-0 z-10 flex items-center gap-2 border-b px-4 py-2 md:gap-4"
        >
            <NuxtLink to="/" class="flex shrink-0 items-center">
                <AppLogo class="h-5 w-auto shrink-0" />
            </NuxtLink>
            <div class="flex md:hidden">
                <DesignMenu
                    :items="toolMenuItems"
                    placement="bottom-start"
                    trigger-icon="tabler:menu-2"
                    :trigger-label="t('toolsMenu')"
                    @select="navigateTo($event)"
                />
            </div>
            <nav class="hidden items-center gap-1 md:flex">
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
            <LanguageSwitcher class="min-w-0" />
        </header>
        <main class="grow">
            <slot />
        </main>
    </div>
</template>
