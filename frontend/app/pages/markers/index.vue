<script lang="ts" setup>
const { t } = useI18n();

useHead({ title: "Markers" });

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({ id: "markers", title: "markers" });
});

const vxId = ref("");

function open() {
    const id = vxId.value.trim();
    if (!id) return;
    navigateTo(`/markers/${id}`);
}
</script>

<template>
    <div class="mx-auto flex h-full max-w-7xl items-center justify-center p-8">
        <div class="flex w-full max-w-sm flex-col items-center gap-6">
            <div class="text-center">
                <Icon
                    name="tabler:bookmarks"
                    class="text-text-muted mb-2 size-10"
                />
                <h1 class="text-heading-3 text-text-default">
                    {{ t("markers.index.title") }}
                </h1>
                <p class="text-text-muted text-sm">
                    {{ t("markers.index.description") }}
                </p>
            </div>
            <form class="flex w-full flex-col gap-2" @submit.prevent="open">
                <DesignInput
                    v-model="vxId"
                    label="VX-ID"
                    required
                    placeholder="VX-123456"
                    leading-icon="tabler:hash"
                />
                <DesignButton
                    type="submit"
                    class="w-full"
                    icon="tabler:arrow-right"
                >
                    {{ t("markers.index.open") }}
                </DesignButton>
            </form>
            <p class="text-text-hint text-caption-1 text-center">
                {{ t("markers.index.demoHint") }}
            </p>
        </div>
    </div>
</template>
