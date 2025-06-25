<script lang="ts" setup>
import { BccButton, BccInput } from "@bcc-code/design-library-vue";

useHead({
    title: "Transcription",
});

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({
        id: "transcription",
        title: "transcription",
    });
});

const transcription = ref<TranscriptionResult>();

const fileName = ref<string>();

const tKey = ref<string>();

const handleFile = (event: Event) => {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (file) {
        fileName.value = file.name;
        transcription.value = undefined;
        segments.value = [];
        const reader = new FileReader();
        reader.onload = (e) => {
            const result = e.target?.result;
            if (result) {
                transcription.value = JSON.parse(result.toString());
                transcription.value!.segments.forEach((s) => {
                    s.text = s.text.trim();
                    s.words = s.words.map((w) => ({
                        ...w,
                        text: w.text.trim(),
                    }));
                });

                segments.value = transcription.value!.segments ?? [];

                tKey.value = generateRandomId();
            }
        };
        reader.readAsText(file);
    }
};

const segments = ref<Segment[]>([]);

const vxId = ref("");

const { deleteMode } = useDeleteMode();

const truncatedFileName = computed(() => {
    if (!fileName.value) return;
    if (fileName.value.length < 30) return fileName.value;

    const first = fileName.value.slice(0, 10);
    const last = fileName.value.slice(-10);
    return [first, last].join("...");
});
</script>

<template>
    <div
        :class="[
            'flex h-screen',
            {
                'border-8 border-red-700': deleteMode,
            },
        ]"
    >
        <div class="flex flex-grow flex-col">
            <div class="flex max-w-80 items-center gap-4">
                <div class="shrink-0">
                    <label for="file-input" class="cursor-pointer">
                        <BccButton class="pointer-events-none">
                            {{
                                fileName && truncatedFileName
                                    ? truncatedFileName
                                    : "Select file"
                            }}
                        </BccButton>
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
                <template v-if="!fileName">
                    <span class="text-sm text-tertiary">or</span>
                    <div class="flex gap-1 rounded-xl bg-neutral-100 p-2">
                        <BccInput
                            v-model="vxId"
                            placeholder="Vidispine-ID"
                            class="min-w-32"
                        />
                        <BccButton
                            @click="navigateTo(`/transcription/${vxId}`)"
                            variant="tertiary"
                        >
                            Go
                        </BccButton>
                    </div>
                </template>
            </div>
            <template v-if="fileName" class="flex gap-4">
                <TranscriptionDownloader
                    :segments="segments"
                    :filename="fileName"
                />
            </template>
            <TranscriptionEditor
                :key="tKey"
                :transcription="transcription"
                :file-name="fileName"
                v-model="segments"
            />
        </div>
    </div>
</template>
