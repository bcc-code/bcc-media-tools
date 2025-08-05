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
                    <p>BMM Upload</p>
                    <p class="text-sm text-neutral-400">
                        Upload localized audio files to BMM
                    </p>
                </UCard>
            </NuxtLink>
            <NuxtLink to="/transcription/" class="aspect-video">
                <UCard class="size-full">
                    <Icon name="tabler:text-recognition" class="mb-2 size-8" />
                    <p>Transcription</p>
                    <p class="text-sm text-neutral-400">
                        Edit and download transcriptions of video files
                    </p>
                </UCard>
            </NuxtLink>
            <NuxtLink v-if="me?.admin" to="/admin/" class="aspect-video">
                <UCard
                    class="size-full"
                    :ui="{ body: 'flex flex-col h-full items-start' }"
                >
                    <Icon name="tabler:settings" class="mb-2 size-8" />
                    <p>Admin</p>
                    <p class="text-sm text-neutral-400">
                        Manage users and settings
                    </p>
                    <UBadge
                        size="sm"
                        variant="outline"
                        color="neutral"
                        class="mt-auto"
                    >
                        Only for Admins
                    </UBadge>
                </UCard>
            </NuxtLink>
        </div>
    </div>
</template>
