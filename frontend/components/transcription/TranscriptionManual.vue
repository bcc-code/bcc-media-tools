<script lang="ts" setup>
import { BccButton, BccModal } from "@bcc-code/design-library-vue";

const open = defineModel<boolean>("open", { required: true });

const keyboardShortcuts = {
    "↑": "transcription.goToPreviousSegment",
    "↓": "transcription.goToNextSegment",
    tab: "transcription.goToNextWord",
};
</script>

<template>
    <BccModal :open="open" @close="open = false" :close-button="false">
        <dl
            class="grid gap-x-4 gap-y-2 py-4"
            style="grid-template-columns: max-content"
        >
            <template v-for="(value, key) in keyboardShortcuts" :key="key">
                <dt>
                    <KeyboardKeys :keys="[key]" />
                </dt>
                <dd class="col-start-2">
                    {{ $t(value) }}
                </dd>
            </template>
        </dl>
    </BccModal>
</template>
