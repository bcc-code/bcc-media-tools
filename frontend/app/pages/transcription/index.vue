<script lang="ts" setup>
import { useMe } from "~/utils/me";

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

const { me } = useMe();

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
                    <UFileUpload
                        v-if="!transcription"
                        accept="application/json"
                        icon="heroicons:document-text"
                        :label="$t('transcription.uploadJsonFileTitle')"
                        :description="
                            $t('transcription.uploadJsonFileDescription')
                        "
                        :interactive="false"
                        layout="list"
                        @update:model-value="handleFile"
                    >
                        <template #actions="{ open }">
                            <UButton
                                :label="$t('transcription.selectFile')"
                                icon="i-lucide-upload"
                                color="neutral"
                                variant="outline"
                                @click="open()"
                            />
                        </template>
                    </UFileUpload>
                </div>
                <template
                    v-if="
                        !fileName && me?.transcription && me.transcription.admin
                    "
                >
                    <USeparator :label="$t('transcription.or')" />
                    <form
                        class="flex w-full flex-col gap-2"
                        @submit.prevent="navigateTo(`/transcription/${vxId}`)"
                    >
                        <UFormField label="VX-ID">
                            <UInput
                                v-model="vxId"
                                required
                                placeholder="VX-123456"
                                class="w-full"
                            />
                        </UFormField>
                        <UButton type="submit" block>
                            {{ $t("transcription.load") }}
                        </UButton>
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
