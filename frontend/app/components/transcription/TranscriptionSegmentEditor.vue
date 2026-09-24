<script lang="ts" setup>
const props = defineProps<{
    segment: Segment;
    focused?: boolean;
}>();

const emit = defineEmits<{
    update: [Segment];
    wordFocus: [Word, Segment];
    toggleDelete: [];
    focusNext: [];
    focusPrevious: [];
}>();

// Keeps the contenteditable uncontrolled while typing: patching the text node
// on every keystroke would reset the caret. Writing only on a real difference
// makes the echo of the user's own edit a no-op, while a value that changed
// elsewhere (row re-created by the virtual list, reset, restored draft) lands.
const vWordText = {
    mounted(el: HTMLElement, binding: { value: string }) {
        el.textContent = binding.value;
    },
    updated(el: HTMLElement, binding: { value: string }) {
        if (el.textContent !== binding.value) {
            el.textContent = binding.value;
        }
    },
};

const handleTextUpdate = (index: number, event: Event) => {
    const text = (event.target as HTMLElement).textContent ?? "";
    emit("update", setWordText(props.segment, index, text));
};

const { deleteMode } = useDeleteMode();
</script>

<template>
    <div
        class="flex items-center px-6 py-4 transition-all ease-out"
        :class="{
            'cursor-pointer hover:bg-red-200 hover:text-red-700': deleteMode,
            'bg-surface-raise opacity-50': segment.deleted,
            'ring-text-default bg-surface-indent ring-2 ring-inset': focused,
        }"
        :tabindex="deleteMode ? 0 : -1"
        @click="deleteMode ? $emit('toggleDelete') : undefined"
        @keydown.enter="deleteMode ? $emit('toggleDelete') : undefined"
        @keydown.space="deleteMode ? $emit('toggleDelete') : undefined"
    >
        <div class="grow">
            <div class="text-text-hint flex gap-2 text-sm tabular-nums">
                <p>{{ formatTime(segment.start) }}</p>
                -
                <p>{{ formatTime(segment.end) }}</p>
            </div>
            <div
                :class="[
                    'relative flex flex-wrap items-center',
                    { 'pointer-events-none': deleteMode },
                ]"
            >
                <span
                    v-for="(w, index) in segment.words"
                    :key="`${segment.uid}:word:${index}`"
                    v-word-text="w.text"
                    contenteditable
                    :tabindex="deleteMode ? -1 : 0"
                    class="focus:border-text-default focus:bg-surface-indent min-w-4 rounded-md border border-transparent px-2 leading-tight focus:outline-none"
                    @input="handleTextUpdate(index, $event)"
                    @focus="$emit('wordFocus', w, segment)"
                    @keydown.down="$emit('focusNext')"
                    @keydown.up="$emit('focusPrevious')"
                />
            </div>
        </div>
        <div v-if="!deleteMode" class="ml-auto">
            <DesignTooltip
                v-if="!segment.deleted"
                :content="$t('transcription.deleteSegment')"
            >
                <DesignButton
                    variant="tertiary"
                    intent="danger"
                    icon="tabler:trash"
                    @click="$emit('toggleDelete')"
                />
            </DesignTooltip>
            <DesignTooltip
                v-else
                :content="$t('transcription.undeleteSegment')"
            >
                <DesignButton
                    variant="tertiary"
                    icon="tabler:refresh"
                    @click="$emit('toggleDelete')"
                />
            </DesignTooltip>
        </div>
    </div>
</template>
