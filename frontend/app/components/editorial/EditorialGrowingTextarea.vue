<script setup lang="ts">
// A field that wraps instead of truncating. The table's titles and comments
// both run past their column, and a reviewer scanning the list needs the whole
// line, so these grow downwards with their content.
//
// The growing is the no-JS recipe: an invisible twin holds the same text at
// the same metrics in the same grid cell, so it sets the height and the
// textarea stretches over it. Nothing measures or resizes, which means the
// height is right on first paint and after a remote edit arrives, not a frame
// later.
//
// Enter commits rather than inserting a newline: these are one-line values in
// a table row. Text that already holds newlines still renders them.
defineProps<{
    placeholder?: string;
    // A title's placeholder stands in for the row's name when it has no title
    // of its own, so it is real content and carries the weight of it. An
    // ordinary hint ("Legg til en kommentar...") stays dim, or it reads as
    // something already written.
    placeholderIsValue?: boolean;
    ariaLabel?: string;
}>();

const model = defineModel<string>({ required: true });
</script>

<template>
    <div class="grid">
        <textarea
            v-model="model"
            rows="1"
            :placeholder="placeholder"
            :aria-label="ariaLabel"
            class="border-border-1 text-body-3 text-text-default ds-focus-ring col-start-1 row-start-1 w-full resize-none overflow-hidden rounded-xl border px-3 py-2"
            :class="
                placeholderIsValue
                    ? 'placeholder:text-text-muted'
                    : 'placeholder:text-text-hint'
            "
            @keydown.enter.prevent="
                ($event.target as HTMLTextAreaElement).blur()
            "
        />
        <!--
            Mirrors the placeholder too, so a row showing one sizes to it
            rather than clipping it at a single line.
        -->
        <div
            aria-hidden="true"
            class="text-body-3 invisible col-start-1 row-start-1 w-full border px-3 py-2 break-words whitespace-pre-wrap"
        >
            {{ model || placeholder }}&nbsp;
        </div>
    </div>
</template>
