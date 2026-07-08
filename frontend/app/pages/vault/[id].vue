<script lang="ts" setup>
import type { VaultItem } from "~~/src/gen/api/v1/api_pb";

const api = useAPI();
const route = useRoute();
const { t } = useI18n();
const config = useRuntimeConfig();
const base = config.public.grpcUrl;

const vxId = computed(() => route.params.id?.toString() ?? "");

const { chips, loading: actionLoading } = useCantemoActions(vxId);

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
const waveformSrc = computed(
    () =>
        `${base}/vault/waveform?vxid=${encodeURIComponent(vxId.value)}&width=1200&height=240&bgcolor=000000&fgcolor=ffffff`,
);

const { canShorts } = usePermissions();

const isVideo = computed(() => item.value?.mediaType === "video");
const isAudio = computed(() => item.value?.mediaType === "audio");
const isImage = computed(() => item.value?.mediaType === "image");

// Try the higher-res preview first; if it 404s (no preview shape on
// Cantemo for this item), fall back to the static thumbnail.
const imageFailed = ref(false);
const imageSrc = computed(() =>
    imageFailed.value ? thumbSrc.value : previewSrc.value,
);

// Vidispine returns 404 if the item hasn't been analyzed yet — fall back to
// the icon in that case.
const waveformFailed = ref(false);

const bigIcon = computed(() => {
    switch (item.value?.mediaType) {
        case "video":
            return "tabler:video";
        case "audio":
            return "tabler:volume";
        case "image":
            return "tabler:photo";
        default:
            return "tabler:file";
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
    <div class="px-6 pt-8 pb-16">
        <div class="mx-auto max-w-7xl">
            <!-- header -->
            <div class="mb-6 flex items-center gap-4">
                <DesignButton
                    icon="tabler:arrow-left"
                    variant="secondary"
                    class="border-border-1 border"
                    @click="navigateTo('/vault/')"
                >
                    {{ t("vault.back") }}
                </DesignButton>
                <h1
                    class="min-w-0 truncate font-mono text-2xl font-semibold tracking-tight"
                >
                    {{ item?.title || vxId }}
                </h1>
            </div>

            <div class="grid grid-cols-[1.4fr_1fr] items-start gap-6">
                <!-- preview -->
                <div
                    class="bg-surface-default border-border-1 overflow-hidden rounded-[14px] border"
                >
                    <video
                        v-if="isVideo"
                        :src="previewSrc"
                        controls
                        class="aspect-video w-full bg-black"
                    />
                    <div
                        v-else-if="isAudio"
                        class="bg-surface-indent text-text-muted flex aspect-video w-full flex-col items-center justify-center gap-6 p-6"
                    >
                        <img
                            v-if="!waveformFailed"
                            :src="waveformSrc"
                            alt=""
                            class="w-full max-w-3xl mix-blend-lighten"
                            @error="waveformFailed = true"
                        />
                        <Icon
                            v-else
                            :name="bigIcon"
                            class="size-14 opacity-40"
                        />
                        <audio
                            :src="previewSrc"
                            controls
                            class="w-full max-w-md"
                        />
                    </div>
                    <img
                        v-else-if="isImage"
                        :src="imageSrc"
                        :alt="item?.title"
                        class="bg-surface-indent aspect-video w-full object-contain"
                        @error="imageFailed = true"
                    />
                    <div
                        v-else
                        class="bg-surface-indent text-text-muted flex aspect-video items-center justify-center"
                    >
                        <Icon :name="bigIcon" class="size-14 opacity-40" />
                    </div>
                </div>

                <!-- summary -->
                <div
                    class="bg-surface-raise gradient-border shadow-resting rounded-2xl p-5"
                >
                    <h3 class="mb-4 text-sm font-semibold">
                        {{ t("vault.itemSummary") }}
                    </h3>
                    <dl class="flex flex-col">
                        <div
                            class="border-border-1 flex items-center justify-between gap-4 border-b py-3"
                        >
                            <dt class="text-text-muted text-[13px]">
                                {{ t("vault.title") }}
                            </dt>
                            <dd class="text-right text-[13px] break-all">
                                {{ item?.title || "—" }}
                            </dd>
                        </div>
                        <div
                            class="border-border-1 flex items-center justify-between gap-4 border-b py-3"
                        >
                            <dt class="text-text-muted text-[13px]">
                                {{ t("vault.uploadDate") }}
                            </dt>
                            <dd class="text-right text-[13px]">
                                {{ item?.added || "—" }}
                            </dd>
                        </div>
                        <div
                            class="border-border-1 flex items-center justify-between gap-4 border-b py-3"
                        >
                            <dt class="text-text-muted text-[13px]">
                                {{ t("vault.size") }}
                            </dt>
                            <dd class="text-right font-mono text-[13px]">
                                {{ item?.size || "—" }}
                            </dd>
                        </div>
                        <div
                            class="flex items-center justify-between gap-4 py-3"
                        >
                            <dt class="text-text-muted text-[13px]">
                                {{ t("vault.length") }}
                            </dt>
                            <dd class="text-right font-mono text-[13px]">
                                {{ lengthLabel }}
                            </dd>
                        </div>
                    </dl>

                    <div
                        v-if="chips.length || (isVideo && canShorts)"
                        class="border-border-1 mt-4 border-t pt-4"
                    >
                        <h3 class="mb-3 text-sm font-semibold">
                            {{ t("vault.actions") }}
                        </h3>
                        <div class="flex flex-wrap gap-2">
                            <DesignButton
                                v-for="chip in chips"
                                :key="chip.name"
                                size="small"
                                variant="secondary"
                                :title="chip.action"
                                :loading="actionLoading === chip.name"
                                :disabled="!!actionLoading"
                                @click="chip.run()"
                            >
                                <span
                                    class="mr-1 inline-block size-1.5 rounded-full"
                                    :style="{ backgroundColor: chip.color }"
                                />
                                {{ chip.name }}
                            </DesignButton>
                            <DesignButton
                                v-if="isVideo && canShorts"
                                size="small"
                                variant="secondary"
                                :title="t('vault.createShort')"
                                @click="
                                    navigateTo({
                                        name: 'shorts-generate',
                                        query: { id: vxId },
                                    })
                                "
                            >
                                <span
                                    class="mr-1 inline-block size-1.5 rounded-full"
                                    :style="{ backgroundColor: '#ec4899' }"
                                />
                                {{ t("vault.createShort") }}
                            </DesignButton>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
