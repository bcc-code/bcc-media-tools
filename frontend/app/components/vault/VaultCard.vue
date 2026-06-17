<script lang="ts" setup>
import type { VaultItem } from "~~/src/gen/api/v1/api_pb";

const props = defineProps<{
    item: VaultItem;
    // Backend base URL (where the /vault/thumbnail proxy lives).
    base: string;
}>();

const { t } = useI18n();

// Optional trick-play fraction (0..1 along the asset) while hovering a video.
const frac = ref<number | null>(null);

// Cantemo doesn't auto-generate `/thumbnailresource` for static images, so the
// thumbnail endpoint 404s for them. Fall back to the preview shape (the actual
// image bytes) for image items only — for video/audio the preview is media,
// not an image, so loading it into <img> would just waste bandwidth.
type ImgStage = "thumbnail" | "preview" | "failed";
const imgStage = ref<ImgStage>("thumbnail");

const thumbSrc = computed(() => {
    let url = `${props.base}/vault/thumbnail?vxid=${encodeURIComponent(props.item.VXID)}`;
    if (frac.value != null) url += `&f=${frac.value}`;
    return url;
});

// /vault/image is a backend-resized JPEG version of the preview shape,
// suitable for grid thumbnails. Detail page still uses /vault/preview for
// full resolution.
const previewSrc = computed(
    () =>
        `${props.base}/vault/image?vxid=${encodeURIComponent(props.item.VXID)}&width=400`,
);

// /vault/waveform proxies Vidispine's pre-rendered waveform PNG. Black bg +
// white fg renders as a transparent waveform when combined with
// `mix-blend-mode: lighten` on a darker card background.
const waveformSrc = computed(
    () =>
        `${props.base}/vault/waveform?vxid=${encodeURIComponent(props.item.VXID)}&width=400&height=160&bgcolor=000000&fgcolor=ffffff`,
);

const imgSrc = computed(() => {
    if (imgStage.value === "failed") return undefined;
    if (props.item.mediaType === "audio") return waveformSrc.value;
    return imgStage.value === "thumbnail" ? thumbSrc.value : previewSrc.value;
});

function onImgError() {
    if (imgStage.value === "thumbnail" && props.item.mediaType === "image") {
        imgStage.value = "preview";
    } else {
        imgStage.value = "failed";
    }
}

const typeIcon = computed(() => {
    switch (props.item.mediaType) {
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

const durationLabel = computed(() => {
    const s = props.item.durationSeconds;
    if (!s) return "";
    const m = Math.floor(s / 60);
    const sec = s % 60;
    return `${String(m).padStart(2, "0")}:${String(sec).padStart(2, "0")}`;
});

// Scrub through thumbnail frames based on the cursor position (trick-play).
// Quantize to a handful of steps to limit thumbnail requests while hovering.
function onMove(e: MouseEvent) {
    if (props.item.mediaType !== "video") return;
    if (imgStage.value === "failed") return;
    const el = e.currentTarget as HTMLElement;
    const rect = el.getBoundingClientRect();
    const f = Math.min(Math.max((e.clientX - rect.left) / rect.width, 0), 1);
    const step = Math.round(f * 12) / 12;
    if (step !== frac.value) frac.value = step;
}

function onLeave() {
    frac.value = null;
}
</script>

<template>
    <NuxtLink
        :to="`/vault/${item.VXID}`"
        class="bg-default border-default hover:border-accented hover:bg-elevated block overflow-hidden rounded-[14px] border transition-colors"
    >
        <!-- thumbnail -->
        <div
            class="bg-muted text-muted relative flex aspect-16/10 items-center justify-center"
            @mousemove="onMove"
            @mouseleave="onLeave"
        >
            <img
                v-if="imgStage !== 'failed'"
                :src="imgSrc"
                :alt="item.title"
                loading="lazy"
                :class="[
                    'h-full w-full object-contain',
                    item.mediaType === 'audio' && 'mix-blend-lighten',
                ]"
                @error="onImgError"
            />
            <UIcon v-else :name="typeIcon" class="size-10 opacity-40" />
            <span
                v-if="durationLabel"
                class="bg-default/70 text-default absolute top-2 left-2 rounded-md px-1.5 py-0.5 font-mono text-[11px]"
            >
                {{ durationLabel }}
            </span>
        </div>
        <!-- meta -->
        <div class="p-3">
            <div class="flex min-w-0 items-center gap-1.5">
                <UIcon :name="typeIcon" class="text-muted size-3.5 shrink-0" />
                <span class="truncate text-[13px] font-medium">{{
                    item.title
                }}</span>
            </div>
            <div class="text-muted mt-2 text-[11px] leading-relaxed">
                <div>{{ t("vault.added") }}: {{ item.added || "—" }}</div>
                <div>
                    {{ t("vault.format") }}:
                    <span class="font-mono">{{ item.format || "—" }}</span>
                </div>
            </div>
        </div>
    </NuxtLink>
</template>
