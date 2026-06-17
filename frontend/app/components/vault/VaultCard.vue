<script lang="ts" setup>
import type { VaultItem } from "~~/src/gen/api/v1/api_pb";

const props = defineProps<{
    item: VaultItem;
    // Backend base URL (where the /vault/thumbnail proxy lives).
    base: string;
}>();

const { t } = useI18n();

// Optional trick-play frame (seconds into the asset) while hovering a video.
const frame = ref<number | null>(null);
const imgFailed = ref(false);

const thumbSrc = computed(() => {
    let url = `${props.base}/vault/thumbnail?vxid=${encodeURIComponent(props.item.VXID)}`;
    if (frame.value != null) url += `&t=${frame.value}`;
    return url;
});

const typeIcon = computed(() => {
    switch (props.item.mediaType) {
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

const durationLabel = computed(() => {
    const s = props.item.durationSeconds;
    if (!s) return "";
    const m = Math.floor(s / 60);
    const sec = s % 60;
    return `${String(m).padStart(2, "0")}:${String(sec).padStart(2, "0")}`;
});

// Scrub through thumbnail frames based on the cursor position (trick-play).
function onMove(e: MouseEvent) {
    if (props.item.mediaType !== "video" || !props.item.durationSeconds) return;
    if (imgFailed.value) return;
    const el = e.currentTarget as HTMLElement;
    const rect = el.getBoundingClientRect();
    const frac = Math.min(Math.max((e.clientX - rect.left) / rect.width, 0), 1);
    const sec = Math.floor(frac * props.item.durationSeconds);
    if (sec !== frame.value) frame.value = sec;
}

function onLeave() {
    frame.value = null;
}
</script>

<template>
    <NuxtLink
        :to="`/vault/${item.VXID}`"
        class="bg-default border-default hover:border-accented hover:bg-elevated block overflow-hidden rounded-[14px] border transition-colors"
    >
        <!-- thumbnail -->
        <div
            class="bg-muted text-muted relative flex aspect-[16/10] items-center justify-center"
            @mousemove="onMove"
            @mouseleave="onLeave"
        >
            <img
                v-if="!imgFailed"
                :src="thumbSrc"
                :alt="item.title"
                loading="lazy"
                class="h-full w-full object-cover"
                @error="imgFailed = true"
            />
            <UIcon v-else :name="typeIcon" class="size-10 opacity-40" />
            <span
                v-if="durationLabel"
                class="bg-default/70 text-default absolute left-2 top-2 rounded-md px-1.5 py-0.5 font-mono text-[11px]"
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
