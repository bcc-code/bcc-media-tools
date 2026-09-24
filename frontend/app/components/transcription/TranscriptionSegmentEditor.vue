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

// Keeps the contenteditable uncontrolled while the user types: patching the
// text node on every keystroke would reset the caret, and the canonical text is
// whitespace-normalised, so it would also fight what is being typed. A value
// that changed elsewhere (row re-created by the virtual list, reset, restored
// draft) still lands, because then the field is not focused.
const vSegmentText = {
    mounted(el: HTMLElement, binding: { value: string }) {
        el.textContent = binding.value;
    },
    updated(el: HTMLElement, binding: { value: string }) {
        if (document.activeElement === el) return;
        if (el.textContent !== binding.value) el.textContent = binding.value;
    },
};

const field = useTemplateRef<HTMLElement>("field");

const handleInput = (event: Event) => {
    const text = (event.target as HTMLElement).textContent ?? "";
    emit("update", setSegmentText(props.segment, text));
};

const handleBlur = () => {
    if (field.value && field.value.textContent !== props.segment.text) {
        field.value.textContent = props.segment.text;
    }
};

/** Character offset of the caret within the field, across split text nodes. */
const caretOffset = (): number => {
    const selection = window.getSelection();
    if (!selection?.anchorNode || !field.value) return 0;
    if (!field.value.contains(selection.anchorNode)) return 0;

    const range = document.createRange();
    range.selectNodeContents(field.value);
    range.setEnd(selection.anchorNode, selection.anchorOffset);
    return range.toString().length;
};

// Seeking follows the caret, so it still lands on the right word now that a
// segment is one field rather than one element per word.
const reportCaretWord = () => {
    const word = wordAtOffset(props.segment.words, caretOffset());
    if (word) emit("wordFocus", word, props.segment);
};

// Pasting is taken over so that markup and line breaks never enter the field.
const handlePaste = (event: ClipboardEvent) => {
    event.preventDefault();

    const selection = window.getSelection();
    if (!field.value || !selection?.rangeCount) return;

    const text = (event.clipboardData?.getData("text/plain") ?? "").replace(
        /\s+/g,
        " ",
    );

    const range = selection.getRangeAt(0);
    range.deleteContents();
    const inserted = document.createTextNode(text);
    range.insertNode(inserted);
    range.setStartAfter(inserted);
    range.collapse(true);
    selection.removeAllRanges();
    selection.addRange(range);

    // Editing the DOM directly does not fire `input`.
    emit(
        "update",
        setSegmentText(props.segment, field.value.textContent ?? ""),
    );
};

const { deleteMode } = useDeleteMode();
</script>

<template>
    <div
        class="flex items-center gap-4 px-6 py-4 transition-all ease-out"
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
        <div class="min-w-0 grow">
            <div class="text-text-hint flex gap-2 text-sm tabular-nums">
                <p>{{ formatTime(segment.start) }}</p>
                -
                <p>{{ formatTime(segment.end) }}</p>
            </div>
            <div
                ref="field"
                v-segment-text="segment.text"
                contenteditable
                role="textbox"
                :tabindex="deleteMode ? -1 : 0"
                :class="[
                    'focus:border-text-default focus:bg-surface-indent min-h-8 rounded-md border border-transparent px-2 py-0.5 leading-tight focus:outline-none',
                    { 'pointer-events-none': deleteMode },
                ]"
                @input="handleInput"
                @blur="handleBlur"
                @focus="reportCaretWord"
                @keyup="reportCaretWord"
                @mouseup="reportCaretWord"
                @keydown.enter.prevent
                @keydown.down="$emit('focusNext')"
                @keydown.up="$emit('focusPrevious')"
                @paste="handlePaste"
            />
        </div>
        <div v-if="!deleteMode" class="shrink-0">
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
