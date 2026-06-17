<script lang="ts" setup>
import type { VaultItem } from "~~/src/gen/api/v1/api_pb";

const api = useAPI();
const route = useRoute();
const { t } = useI18n();
const config = useRuntimeConfig();
const base = config.public.grpcUrl;

const vxId = computed(() => route.params.id?.toString() ?? "");

const item = ref<VaultItem>();
const loading = ref(true);

const pageTitle = computed(() => item.value?.title || "Vault");
useHead({ title: pageTitle });

onMounted(async () => {
    try {
        const res = await api.getVaultItem({ VXID: vxId.value });
        item.value = res.item;
    } catch {
        // ignore — the summary simply shows placeholders
    } finally {
        loading.value = false;
    }
});

const previewSrc = computed(
    () => `${base}/vault/preview?vxid=${encodeURIComponent(vxId.value)}`,
);
const thumbSrc = computed(
    () => `${base}/vault/thumbnail?vxid=${encodeURIComponent(vxId.value)}`,
);

const isPlayable = computed(
    () =>
        item.value?.mediaType === "video" || item.value?.mediaType === "audio",
);
const isImage = computed(() => item.value?.mediaType === "image");

const bigIcon = computed(() => {
    switch (item.value?.mediaType) {
        case "video":
            return "i-lucide-video";
        case "audio":
            return "i-lucide-volume-2";
        case "image":
            return "i-lucide-image";
        default:
            return "i-lucide-file";
    }
});

const lengthLabel = computed(() => {
    const s = item.value?.durationSeconds ?? 0;
    if (!s) return "—";
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = s % 60;
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${pad(h)}:${pad(m)}:${pad(sec)}`;
});
</script>

<template>
    <div class="px-6 pb-16 pt-8">
        <div class="mx-auto max-w-[920px]">
            <!-- header -->
            <div class="mb-6 flex items-center gap-4">
                <UButton
                    icon="i-lucide-arrow-left"
                    color="neutral"
                    variant="outline"
                    @click="navigateTo('/vault/')"
                >
                    {{ t("vault.back") }}
                </UButton>
                <h1
                    class="truncate font-mono text-2xl font-semibold tracking-tight"
                >
                    {{ item?.title || vxId }}
                </h1>
            </div>

            <div class="grid grid-cols-[1.4fr_1fr] items-start gap-6">
                <!-- preview -->
                <div
                    class="bg-default border-default overflow-hidden rounded-[14px] border"
                >
                    <video
                        v-if="isPlayable"
                        :src="previewSrc"
                        controls
                        class="aspect-video w-full bg-black"
                    />
                    <img
                        v-else-if="isImage"
                        :src="thumbSrc"
                        :alt="item?.title"
                        class="bg-muted aspect-video w-full object-contain"
                    />
                    <div
                        v-else
                        class="bg-muted text-muted flex aspect-video items-center justify-center"
                    >
                        <UIcon :name="bigIcon" class="size-14 opacity-40" />
                    </div>
                </div>

                <!-- summary -->
                <div
                    class="bg-elevated border-default rounded-[14px] border p-5"
                >
                    <h3 class="mb-4 text-sm font-semibold">
                        {{ t("vault.itemSummary") }}
                    </h3>
                    <dl class="flex flex-col">
                        <div
                            class="border-default flex items-center justify-between gap-4 border-b py-3"
                        >
                            <dt class="text-muted text-[13px]">
                                {{ t("vault.title") }}
                            </dt>
                            <dd class="break-all text-right text-[13px]">
                                {{ item?.title || "—" }}
                            </dd>
                        </div>
                        <div
                            class="border-default flex items-center justify-between gap-4 border-b py-3"
                        >
                            <dt class="text-muted text-[13px]">
                                {{ t("vault.uploadDate") }}
                            </dt>
                            <dd class="text-right text-[13px]">
                                {{ item?.added || "—" }}
                            </dd>
                        </div>
                        <div
                            class="border-default flex items-center justify-between gap-4 border-b py-3"
                        >
                            <dt class="text-muted text-[13px]">
                                {{ t("vault.size") }}
                            </dt>
                            <dd class="text-right font-mono text-[13px]">
                                {{ item?.size || "—" }}
                            </dd>
                        </div>
                        <div
                            class="flex items-center justify-between gap-4 py-3"
                        >
                            <dt class="text-muted text-[13px]">
                                {{ t("vault.length") }}
                            </dt>
                            <dd class="text-right font-mono text-[13px]">
                                {{ lengthLabel }}
                            </dd>
                        </div>
                    </dl>
                </div>
            </div>
        </div>
    </div>
</template>
