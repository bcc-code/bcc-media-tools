<script setup lang="ts">
useHead({
    title: "Home",
});

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({
        id: "index",
        title: "index",
    });
});

const { enabledTools } = useTools();
</script>

<template>
    <div class="mx-auto max-w-7xl px-4">
        <div class="mt-8 grid min-w-1/2 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <NuxtLink
                v-for="tool in enabledTools"
                :key="tool.to"
                :to="tool.to"
                class="aspect-video"
            >
                <div
                    class="gradient-border bg-surface-raise shadow-resting ease-out-expo hover:shadow-floating relative flex size-full flex-col items-start rounded-2xl p-6 transition duration-300 hover:-translate-y-1"
                >
                    <Icon
                        :name="tool.icon"
                        class="text-text-default mb-2 text-lg"
                    />
                    <p class="text-text-default">{{ tool.label }}</p>
                    <p class="text-text-hint text-sm">
                        {{ tool.description }}
                    </p>
                    <DesignBadge
                        v-if="tool.to.startsWith('/admin')"
                        class="mt-auto"
                    >
                        {{ $t("tools.admin.badge") }}
                    </DesignBadge>
                </div>
            </NuxtLink>
        </div>
    </div>
</template>
