<script lang="ts" setup>
const api = useAPI();
const toaster = useToast();

const loading = ref(false);
const confirm = ref(false);

async function finish() {
    if (loading.value) return;
    loading.value = true;
    try {
        const res = await api.finishLiveIngest({});
        const names = res.finished
            .map((f) => f.filename)
            .filter(Boolean)
            .join(", ");
        toaster.create({
            title: t("liveIngest.finished"),
            description: names || undefined,
            type: "success",
        });
        confirm.value = false;
    } catch (err) {
        toaster.create({
            title: t("liveIngest.finishFailed"),
            description: (err as Error)?.message,
            type: "error",
        });
    } finally {
        loading.value = false;
    }
}

const { t } = useI18n();
</script>

<template>
    <div class="mx-auto flex w-full max-w-2xl flex-col gap-6 p-8">
        <header>
            <h1 class="text-heading-3 text-text-default">
                {{ $t("liveIngest.title") }}
            </h1>
            <p class="text-text-muted text-sm">
                {{ $t("liveIngest.description") }}
            </p>
        </header>

        <div>
            <DesignButton
                icon="tabler:player-stop"
                :loading="loading"
                @click="confirm = true"
            >
                {{ $t("liveIngest.finish") }}
            </DesignButton>
        </div>

        <DesignDialog
            v-model:open="confirm"
            :title="$t('liveIngest.confirmTitle')"
            :description="$t('liveIngest.confirmMessage')"
        >
            <div class="flex w-full justify-end gap-2">
                <DesignButton variant="tertiary" @click="confirm = false">
                    {{ $t("liveIngest.cancel") }}
                </DesignButton>
                <DesignButton
                    variant="primary"
                    :loading="loading"
                    @click="finish"
                >
                    {{ $t("liveIngest.confirmSubmit") }}
                </DesignButton>
            </div>
        </DesignDialog>
    </div>
</template>
