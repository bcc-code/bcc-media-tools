<script setup lang="ts">
const props = defineProps<{
    styles: string[];
    // Backend base URL (where the /subtitle-style-preview handler lives).
    base: string;
}>();

const model = defineModel<string>({ default: "" });

// Styles whose preview image failed to load (none uploaded yet) — fall back to a
// placeholder instead of a broken image.
const failed = reactive<Record<string, boolean>>({});

const previewUrl = (style: string) =>
    `${props.base}/subtitle-style-preview?name=${encodeURIComponent(style)}`;

// Display label without the file extension (e.g. "Bold.ass" -> "Bold").
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
    <div v-else class="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <button
            v-for="style in styles"
            :key="style"
            type="button"
            class="ds-focus-ring relative flex flex-col overflow-hidden rounded-xl border text-left"
            :class="
                model === style
                    ? 'border-primary-default ring-primary-default ring-2'
                    : 'border-border-1 hover:border-border-2'
            "
            @click="model = style"
        >
            <div class="bg-surface-indent aspect-video w-full">
                <img
                    v-if="!failed[style]"
                    :src="previewUrl(style)"
                    :alt="label(style)"
                    class="h-full w-full object-cover"
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
            <span
                class="text-caption-1 text-text-default truncate px-2 py-1.5"
                >{{ label(style) }}</span
            >
            <span
                v-if="model === style"
                class="bg-primary-default text-on-primary absolute top-1.5 right-1.5 flex size-5 items-center justify-center rounded-full"
            >
                <Icon name="tabler:check" class="size-3.5" />
            </span>
        </button>
    </div>
</template>
