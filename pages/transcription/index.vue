<script lang="ts" setup>
import { BccButton, BccInput } from "@bcc-code/design-library-vue";

const transcription = ref<TranscriptionResult>();

const fileName = ref<string>();

const handleFile = (event: Event) => {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (file) {
        fileName.value = file.name;
        const reader = new FileReader();
        reader.onload = (e) => {
            const result = e.target?.result;
            if (result) {
                transcription.value = JSON.parse(result.toString());
                transcription.value?.segments.forEach((s) => {
                    s.text = s.text.trim();
                    s.words = s.words.map((w) => ({
                        ...w,
                        text: w.text.trim(),
                    }));
                });

                segments.value = transcription.value?.segments ?? [];
            }
        };
        reader.readAsText(file);
    }
};

const segments = ref<Segment[]>([]);

const download = () => {
    downloadTranscription(segments.value, fileName.value!);
};

const vxId = ref("");
</script>

<template>
    <div class="flex h-screen">
        <div class="flex flex-grow flex-col">
            <TranscriptionEditor
                :transcription="transcription"
                :file-name="fileName!"
                v-model="segments"
            >
                <template #actions>
                    <div>
                        <label for="file-input" class="cursor-pointer">
                            <BccButton class="pointer-events-none">{{
                                fileName ?? "Select file"
                            }}</BccButton>
                        </label>
                        <input
                            id="file-input"
                            hidden
                            type="file"
                            placeholder="File here"
                            accept="application/json"
                            @input="handleFile"
                        />
                    </div>
                    <BccButton @click="download">Download</BccButton>
                    <div class="flex">
                        <input
                            v-model="vxId"
                            class="h-full rounded-l-lg bg-black px-2 text-base"
                            placeholder="Vidispine-ID"
                        />
                        <button
                            class="rounded-r-lg bg-blue-500 px-2"
                            @click="navigateTo(`/transcription/${vxId}`)"
                        >
                            Go
                        </button>
                    </div>
                </template>
            </TranscriptionEditor>
        </div>
    </div>
</template>
