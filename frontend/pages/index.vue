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

const { me } = useMe();
</script>

<template>
    <div class="flex w-full justify-center">
        <div class="grid min-w-1/2 gap-4 p-8 md:grid-cols-3">
            <NuxtLink to="/upload/bmm/" class="aspect-video">
                <UCard class="size-full">
                    <Icon name="tabler:upload" class="mb-2 size-8" />
                    <p>{{ $t("home.bmmUpload.title") }}</p>
                    <p class="text-sm text-neutral-400">
                        {{ $t("home.bmmUpload.description") }}
                    </p>
                </UCard>
            </NuxtLink>
            <NuxtLink to="/transcription/" class="aspect-video">
                <UCard class="size-full">
                    <Icon name="tabler:text-recognition" class="mb-2 size-8" />
                    <p>{{ $t("home.transcription.title") }}</p>
                    <p class="text-sm text-neutral-400">
                        {{ $t("home.transcription.description") }}
                    </p>
                </UCard>
            </NuxtLink>
            <NuxtLink v-if="me?.admin" to="/admin/" class="aspect-video">
                <UCard
                    class="size-full"
                    :ui="{ body: 'flex flex-col h-full items-start' }"
                >
                    <Icon name="tabler:settings" class="mb-2 size-8" />
                    <p>{{ $t("home.admin.title") }}</p>
                    <p class="text-sm text-neutral-400">
                        {{ $t("home.admin.description") }}
                    </p>
                    <UBadge
                        size="sm"
                        variant="outline"
                        color="neutral"
                        class="mt-auto"
                    >
                        {{ $t("home.admin.badge") }}
                    </UBadge>
                </UCard>
            </NuxtLink>
        </div>
    </div>
</template>
