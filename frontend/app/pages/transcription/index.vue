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

const transcription = ref<TranscriptionResult>();

const fileName = ref<string>();

const tKey = ref<string>();

// DesignFileUpload uses a File[] model; this page only takes the first file.
const uploadFiles = ref<File[]>([]);
watch(uploadFiles, (files) => handleFile(files[0]));

const handleFile = (file: File | null | undefined) => {
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

const { transcriptionAdmin } = usePermissions();

function setSegments(s: Segment[]) {
    segments.value = s;
    if (!transcription.value) return;
    transcription.value.segments = s;
}
</script>

<template>
    <div
        :class="[
            'mx-auto flex h-screen max-w-7xl p-8',
            {
                'border-8 border-red-700': deleteMode,
            },
        ]"
    >
        <div class="flex grow flex-col">
            <div
                class="mx-auto flex w-full max-w-sm flex-col items-center gap-4"
            >
                <div class="w-full shrink-0">
                    <DesignFileUpload
                        v-if="!transcription"
                        v-model="uploadFiles"
                        accept="application/json"
                        icon="tabler:file-text"
                        :label="$t('transcription.uploadJsonFileTitle')"
                        :description="
                            $t('transcription.uploadJsonFileDescription')
                        "
                    />
                </div>
                <template v-if="!fileName && transcriptionAdmin">
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
                :key="tKey"
                :transcription="transcription"
                :file-name="fileName"
                v-model="segments"
                @update-segments="(s) => setSegments(s)"
            />
        </div>
    </div>
</template>
