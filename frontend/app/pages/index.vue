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
    <UContainer>
        <div class="mt-8 grid min-w-1/2 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <NuxtLink
                v-for="tool in enabledTools"
                :key="tool.to"
                :to="tool.to"
                class="aspect-video"
            >
                <UCard
                    :ui="{ body: 'flex flex-col h-full items-start' }"
                    class="ease-out-expo relative size-full shadow-xs transition duration-300 hover:-translate-y-1 hover:shadow-md"
                >
                    <Icon :name="tool.icon" class="mb-2 text-lg" />
                    <p>{{ tool.label }}</p>
                    <p class="text-sm text-neutral-400">
                        {{ tool.description }}
                    </p>
                    <UBadge
                        v-if="tool.to.startsWith('/admin')"
                        size="sm"
                        variant="outline"
                        color="neutral"
                        class="mt-auto"
                    >
                        {{ $t("tools.admin.badge") }}
                    </UBadge>
                </UCard>
            </NuxtLink>
        </div>
    </UContainer>
</template>
