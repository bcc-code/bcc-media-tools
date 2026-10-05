<script setup lang="ts">
const props = defineProps<{
    elementWidth: number;
    elementHeight: number;
    videoWidth: number;
    videoHeight: number;
}>();

const guide = computed(() =>
    cropGuideBox(
        props.elementWidth,
        props.elementHeight,
        props.videoWidth,
        props.videoHeight,
    ),
);

const px = (n: number) => `${n}px`;

// Everything the short would throw away, as two bands either side of the crop.
const discarded = computed(() => {
    const g = guide.value;
    if (!g) return [];
    const pictureRight = g.picture.left + g.picture.width;
    const cropRight = g.crop.left + g.crop.width;
    return [
        { left: g.picture.left, width: g.crop.left - g.picture.left },
        { left: cropRight, width: pictureRight - cropRight },
    ].filter((band) => band.width > 0.5);
});
</script>

<template>
    <div v-if="guide" class="pointer-events-none absolute inset-0">
        <div
            v-for="(band, i) in discarded"
            :key="i"
            class="absolute bg-black/55"
            :style="{
                left: px(band.left),
                top: px(guide.picture.top),
                width: px(band.width),
                height: px(guide.picture.height),
            }"
        />
        <div
            class="absolute border-2 border-dashed border-white/70"
            :style="{
                left: px(guide.crop.left),
                top: px(guide.crop.top),
                width: px(guide.crop.width),
                height: px(guide.crop.height),
            }"
        />
    </div>
</template>
