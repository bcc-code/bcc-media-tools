<script lang="ts" setup>
const props = defineProps<{
    segment: Segment;
}>();

const emit = defineEmits<{
    update: [Segment];
}>();

const words = ref(props.segment.words.map((w) => ({ ...w })));

watch(words.value, (v) => {
    emit("update", {
        ...props.segment,
        words: v,
        text: v.map((w) => w.text).join(" "),
    });
});

const handleTextUpdate = (index: number, event: Event) => {
    const target = event.target as HTMLSpanElement;
    const arr = words.value;
    arr[index].text = target.innerText;
    words.value = arr;
};

const secondsToTimestamp = (seconds: number) => {
    const hour = Math.floor(seconds / 3600);
    const minute = Math.floor((seconds % 3600) / 60);
    const second = Math.floor(seconds % 60);
    return `${hour.toString().padStart(2, "0")}:${minute.toString().padStart(2, "0")}:${second.toString().padStart(2, "0")}`;
};
</script>

<template>
    <div class="flex flex-col">
        <div class="flex gap-2 text-sm opacity-50">
            <p>{{ secondsToTimestamp(segment.start) }}</p>
            --
            <p>{{ secondsToTimestamp(segment.end) }}</p>
        </div>
        <div class="flex">
            <span
                contenteditable
                v-for="(w, index) in segment.words"
                @input="handleTextUpdate(index, $event)"
                class="rounded-lg p-1 transition duration-75 focus:bg-slate-800 focus:outline-none"
                >{{ w.text }}</span
            >
        </div>
    </div>
</template>
