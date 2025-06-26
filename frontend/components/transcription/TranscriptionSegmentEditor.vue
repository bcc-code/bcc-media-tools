<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";

const props = defineProps<{
    segment: Segment;
    deleted: boolean;
}>();

const emit = defineEmits<{
    update: [Segment];
    wordFocus: [Word, Segment];
    toggleDelete: [];
    focusNext: [];
    focusPrevious: [];
    addBefore: [];
    addAfter: [];
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

function addWordAt(index: number) {
    const arr = [...words.value];
    arr.splice(index, 0, {
        text: "",
        start: 0,
        end: 0,
        confidence: 0,
    });
    console.log(arr);
    words.value = arr;
}

const hovering = ref(false);

const { deleteMode } = useDeleteMode();
</script>

<template>
    <div
        class="flex items-center px-6 py-4 transition-all ease-out"
        :class="{
            'cursor-pointer hover:bg-red-200 hover:text-red-700': deleteMode,
            'bg-neutral-200 opacity-50': deleted,
        }"
        :tabindex="deleteMode ? 0 : -1"
        @click="deleteMode ? $emit('toggleDelete') : undefined"
        @keydown.enter="deleteMode ? $emit('toggleDelete') : undefined"
        @keydown.space="deleteMode ? $emit('toggleDelete') : undefined"
        @mouseenter="hovering = true"
        @mouseleave="hovering = false"
    >
        <div class="grow">
            <div class="flex gap-2 text-sm opacity-50">
                <p>{{ secondsToTimestamp(segment.start) }}</p>
                -
                <p>{{ secondsToTimestamp(segment.end) }}</p>
            </div>
            <div :class="{ 'pointer-events-none': deleteMode }">
                <TransitionGroup
                    tag="div"
                    class="relative flex flex-wrap items-center"
                    leave-active-class="transition duration-200 ease-out absolute"
                    leave-to-class="opacity-0 scale-0 rotate-180"
                    enter-active-class="transition duration-200 ease-out"
                    enter-from-class="opacity-0 scale-0 rotate-180"
                    move-class="transition duration-200 ease-out"
                >
                    <span
                        v-for="(w, index) in segment.words"
                        :key="`${w.start}-${w.end}`"
                        contenteditable
                        :tabindex="deleteMode ? -1 : 0"
                        class="rounded-md border border-transparent px-2 leading-tight focus:border-gray-900 focus:bg-gray-100 focus:outline-none"
                        @input="handleTextUpdate(index, $event)"
                        @focus="$emit('wordFocus', w, segment)"
                        @keydown.down="$emit('focusNext')"
                        @keydown.up="$emit('focusPrevious')"
                    >
                        {{ w.text }}
                    </span>
                </TransitionGroup>
            </div>
        </div>
        <div v-if="!deleteMode" class="ml-auto">
            <BccButton
                v-if="!deleted"
                context="danger"
                variant="tertiary"
                size="sm"
                :title="$t('transcription.deleteSegment')"
                @click="$emit('toggleDelete')"
            >
                <Icon name="heroicons:trash" />
            </BccButton>
            <BccButton
                v-else
                variant="secondary"
                size="sm"
                :title="$t('transcription.undeleteSegment')"
                @click="$emit('toggleDelete')"
            >
                <Icon name="heroicons:arrow-path" />
            </BccButton>
        </div>
    </div>
</template>
