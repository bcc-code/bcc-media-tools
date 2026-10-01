<script lang="ts" setup>
const open = defineModel<boolean>("open", { required: true });

const keyboardShortcuts = {
    "↑": "transcription.goToPreviousSegment",
    "↓": "transcription.goToNextSegment",
    tab: "transcription.goToNextSegmentField",
};

// The gestures that make marking a long stretch — an intro, a song, a tail of
// dead air — one action instead of a hundred clicks. They are not discoverable
// on their own, so they are spelled out here.
const deleteModeShortcuts = {
    click: "transcription.manualClickRow",
    drag: "transcription.manualDragRows",
    "shift + click": "transcription.manualShiftClick",
    "ctrl/⌘ + z": "transcription.manualUndo",
};
</script>

<template>
    <DesignDialog v-model:open="open">
        <div class="flex flex-col gap-6">
            <section
                v-for="group in [
                    {
                        title: 'transcription.manualKeyboard',
                        shortcuts: keyboardShortcuts,
                    },
                    {
                        title: 'transcription.manualDeleteMode',
                        shortcuts: deleteModeShortcuts,
                    },
                ]"
                :key="group.title"
            >
                <h3 class="text-title-3 text-text-muted mb-3">
                    {{ $t(group.title) }}
                </h3>
                <dl
                    class="grid items-center gap-x-4 gap-y-2"
                    style="grid-template-columns: max-content"
                >
                    <template
                        v-for="(value, key) in group.shortcuts"
                        :key="key"
                    >
                        <dt>
                            <KeyboardKeys :keys="[key]" />
                        </dt>
                        <dd class="col-start-2">
                            {{ $t(value) }}
                        </dd>
                    </template>
                </dl>
            </section>
        </div>
    </DesignDialog>
</template>
