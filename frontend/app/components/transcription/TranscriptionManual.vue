<script lang="ts" setup>
const open = defineModel<boolean>("open", { required: true });

const keyboardShortcuts = {
    "↑": "transcription.goToPreviousSegment",
    "↓": "transcription.goToNextSegment",
    tab: "transcription.goToNextWord",
};
</script>

<template>
    <DesignDialog v-model:open="open">
        <dl
            class="grid gap-x-4 gap-y-2"
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
    </DesignDialog>
</template>
