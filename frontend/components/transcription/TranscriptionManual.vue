<script lang="ts" setup>
import { BccModal } from "@bcc-code/design-library-vue";

const open = defineModel<boolean>("open", { required: true });

const keyboardShortcuts = {
    "↑": "transcription.goToPreviousSegment",
    "↓": "transcription.goToNextSegment",
    tab: "transcription.goToNextWord",
};
</script>

<template>
    <BccModal :open="open" @close="(open = false)">
        <dl
            class="grid gap-x-4 gap-y-2 py-4"
            style="grid-template-columns: max-content"
        >
            <template v-for="(value, key) in keyboardShortcuts" :key="key">
                <dt>
                    <kbd
                        class="aspect-square rounded border border-b-4 border-gray-400 bg-white px-2 py-0.5 text-sm text-gray-500 shadow-sm"
                    >
                        {{ key }}
                    </kbd>
                </dt>
                <dd class="col-start-2">
                    {{ $t(value) }}
                </dd>
            </template>
        </dl>
    </BccModal>
</template>
