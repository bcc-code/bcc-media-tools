<script lang="ts" setup>
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

const fileName = ref<string>();

const tKey = ref<string>();

const segments = ref<Segment[]>([]);

// DesignFileUpload uses a File[] model; this page only takes the first file.
const uploadFiles = ref<File[]>([]);
watch(uploadFiles, (files) => handleFile(files[0]));

const handleFile = (file: File | null | undefined) => {
    if (!file) return;

    fileName.value = file.name;
    segments.value = [];

    const reader = new FileReader();
    reader.onload = (e) => {
        const result = e.target?.result;
        if (!result) return;

        const parsed = JSON.parse(result.toString()) as TranscriptionResult;
        segments.value = withUids(
            (parsed.segments ?? []).map((s) => ({
                ...s,
                text: s.text.trim(),
                words: s.words.map((w) => ({ ...w, text: w.text.trim() })),
            })),
        );

        tKey.value = generateRandomId();
    };
    reader.readAsText(file);
};

const vxId = ref("");

const { deleteMode } = useDeleteMode();

const { isTranscriptionAdmin } = usePermissions();
</script>

<template>
    <div
        :class="[
            'mx-auto flex h-[calc(100dvh-var(--header-height))] max-w-7xl overflow-hidden p-8',
            {
                'border-8 border-red-700': deleteMode,
            },
        ]"
    >
        <div class="flex min-h-0 grow flex-col">
            <div
                class="mx-auto flex w-full max-w-sm shrink-0 flex-col items-center gap-4"
            >
                <div class="w-full shrink-0">
                    <DesignFileUpload
                        v-if="!segments.length"
                        v-model="uploadFiles"
                        accept="application/json"
                        icon="tabler:file-text"
                        :label="$t('transcription.uploadJsonFileTitle')"
                        :description="
                            $t('transcription.uploadJsonFileDescription')
                        "
                    />
                </div>
                <template v-if="!fileName && isTranscriptionAdmin">
                    <div
                        class="text-text-hint text-caption-1 flex w-full items-center gap-3"
                    >
                        <span class="bg-border-1 h-px flex-1" />
                        {{ $t("transcription.or") }}
                        <span class="bg-border-1 h-px flex-1" />
                    </div>
                    <form
                        class="flex w-full flex-col gap-2"
                        @submit.prevent="navigateTo(`/transcription/${vxId}`)"
                    >
                        <DesignInput
                            v-model="vxId"
                            label="VX-ID"
                            required
                            placeholder="VX-123456"
                        />
                        <DesignButton type="submit" class="w-full">
                            {{ $t("transcription.load") }}
                        </DesignButton>
                    </form>
                </template>
                <TranscriptionDownloader
                    v-if="fileName"
                    :segments="segments"
                    :filename="fileName"
                />
            </div>
            <TranscriptionEditor
                v-if="segments.length"
                :key="tKey"
                v-model="segments"
                class="min-h-0 flex-1 overflow-auto"
            />
        </div>
    </div>
</template>
