<script lang="ts" setup>
const open = defineModel<boolean>("open", { required: true });

const keyboardShortcuts = {
    "↑": "transcription.goToPreviousSegment",
    "↓": "transcription.goToNextSegment",
    tab: "transcription.goToNextWord",
};
</script>

<template>
    <UModal v-model:open="open" :close="false">
        <template #body>
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
        </template>
    </UModal>
</template>
