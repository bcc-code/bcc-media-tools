<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";
import TrashIcon from "./TrashIcon.vue";

const props = defineProps<{
    segment: Segment;
    deleted: boolean;
}>();

const emit = defineEmits<{
    update: [Segment];
    wordFocus: [Word];
    toggleDelete: [];
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

const { deleteMode } = useDeleteMode();
</script>

<template>
    <div
        class="flex items-center px-6 py-4 transition-all ease-out"
        :class="{
            'cursor-pointer  hover:bg-red-200 hover:text-red-700 ': deleteMode,
            'bg-neutral-200 opacity-50': deleted,
        }"
        @click="deleteMode ? $emit('toggleDelete') : undefined"
    >
        <div>
            <div class="flex gap-2 text-sm opacity-50">
                <p>{{ secondsToTimestamp(segment.start) }}</p>
                -
                <p>{{ secondsToTimestamp(segment.end) }}</p>
            </div>
            <div :class="{ 'pointer-events-none': deleteMode }">
                <div class="flex flex-wrap">
                    <span
                        contenteditable
                        v-for="(w, index) in segment.words"
                        @input="handleTextUpdate(index, $event)"
                        class="rounded-md border border-transparent px-1 focus:border-neutral-500 focus:bg-neutral-200 focus:outline-none"
                        @focus="$emit('wordFocus', w)"
                    >
                        {{ w.text }}
                    </span>
                </div>
            </div>
        </div>
        <div class="ml-auto">
            <BccButton
                v-if="!deleted"
                context="danger"
                size="sm"
                @click="$emit('toggleDelete')"
            >
                <Icon name="heroicons:trash" />
            </BccButton>
            <BccButton
                v-else
                variant="secondary"
                size="sm"
                @click="$emit('toggleDelete')"
            >
                <Icon name="heroicons:arrow-path" />
            </BccButton>
        </div>
    </div>
</template>
