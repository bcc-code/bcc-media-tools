<script setup lang="ts">
const props = defineProps<{
    styles: string[];
    base: string;
}>();

const model = defineModel<string>({ default: "" });

const failed = reactive<Record<string, boolean>>({});

// True aspect ratio per style, read from the loaded image; used by the lightbox.
const ratios = reactive<Record<string, string>>({});

const onImageLoad = (style: string, e: Event) => {
    const img = e.target as HTMLImageElement;
    if (img.naturalWidth && img.naturalHeight) {
        ratios[style] = `${img.naturalWidth} / ${img.naturalHeight}`;
    }
};

const frameStyle = (style: string) => ({
    aspectRatio: ratios[style] ?? "16 / 9",
});

const enlarged = ref<string | null>(null);
const lightboxOpen = computed({
    get: () => enlarged.value !== null,
    set: (open: boolean) => {
        if (!open) enlarged.value = null;
    },
});

const previewUrl = (style: string) =>
    `${props.base}/subtitle-style-preview?name=${encodeURIComponent(style)}`;

const label = (style: string) => style.replace(/\.[^./]+$/, "");
</script>

<template>
    <div
        v-if="styles.length === 0"
        class="border-border-1 text-text-muted flex flex-col items-center gap-2 rounded-xl border border-dashed px-4 py-8 text-center"
    >
        <Icon name="tabler:photo-off" class="text-text-hint size-6" />
        <p class="text-caption-1">{{ $t("vbExport.noSubtitleStyles") }}</p>
    </div>
    <div v-else class="grid grid-cols-2 items-start gap-3 sm:grid-cols-3">
        <div v-for="style in styles" :key="style" class="relative">
            <button
                type="button"
                class="ds-focus-ring shadow-resting gradient-border bg-surface-default flex w-full flex-col overflow-hidden rounded-xl text-left"
                :class="
                    model === style ? 'ring-primary-default ring-2' : undefined
                "
                @click="model = style"
            >
                <div class="bg-surface-indent aspect-video w-full">
                    <img
                        v-if="!failed[style]"
                        :src="previewUrl(style)"
                        :alt="label(style)"
                        class="h-full w-full object-contain"
                        @load="onImageLoad(style, $event)"
                        @error="failed[style] = true"
                    />
                    <div
                        v-else
                        class="flex h-full w-full items-center justify-center"
                    >
                        <Icon
                            name="tabler:photo-off"
                            class="text-text-hint size-6"
                        />
                    </div>
                </div>
                <p class="text-caption-1 text-text-default truncate px-3 py-2">
                    {{ label(style) }}
                </p>
                <span
                    v-if="model === style"
                    class="bg-primary-default text-on-primary ring-surface-raise shadow-resting absolute top-1.5 right-1.5 flex size-5 items-center justify-center rounded-full ring-2"
                >
                    <Icon name="tabler:check" class="size-3.5" />
                </span>
            </button>
            <DesignTooltip
                v-if="!failed[style]"
                :content="$t('vbExport.enlargePreview')"
            >
                <DesignButton
                    variant="secondary"
                    size="small"
                    icon="tabler:zoom-in"
                    class="text-title-3! absolute! top-1.5! left-1.5! size-6! gap-0! rounded-full! p-0!"
                    @click="enlarged = style"
                />
            </DesignTooltip>
        </div>
    </div>

    <DesignDialog
        v-model:open="lightboxOpen"
        size="xl"
        :title="enlarged ? label(enlarged) : undefined"
    >
        <div
            v-if="enlarged"
            class="bg-surface-indent max-h-[70vh] w-full overflow-hidden rounded-xl"
            :style="frameStyle(enlarged)"
        >
            <img
                :src="previewUrl(enlarged)"
                :alt="label(enlarged)"
                class="h-full w-full object-contain"
            />
        </div>
    </DesignDialog>
</template>
