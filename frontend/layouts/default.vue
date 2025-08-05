<script setup lang="ts">
const { me } = useMe();

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

const route = useRoute();
const items = computed(() => {
    return [
        {
            label: "BMM Upload",
            icon: "tabler:upload",
            to: "/upload/bmm",
        },
        {
            label: "Transcription",
            icon: "tabler:text-recognition",
            to: "/transcription",
            active: route.path.includes("/transcription"),
        },
        {
            label: "Admin",
            icon: "tabler:settings",
            enabled: me.value?.admin,
            to: "/admin",
        },
    ].filter((x) => x.enabled != false);
});
</script>

<template>
    <div class="flex grow-1 flex-col">
        <header
            ref="header"
            class="sticky top-0 z-10 flex items-center gap-8 border-b border-neutral-300 bg-white px-4 dark:border-neutral-700 dark:bg-neutral-900"
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
            <UNavigationMenu orientation="horizontal" highlight :items />
            <ThemeSwitch class="ml-auto" />
        </header>
        <main class="grow-1">
            <slot />
        </main>
    </div>
</template>
