<script setup lang="ts">
import { FileUpload } from "@ark-ui/vue";

interface Props {
    multiple?: boolean;
    accept?: string;
    label?: string;
    description?: string;
    icon?: string;
}

const props = withDefaults(defineProps<Props>(), {
    accept: undefined,
    label: undefined,
    description: undefined,
    icon: "tabler:upload",
});

const model = defineModel<File[]>({ default: () => [] });

const maxFiles = computed(() => (props.multiple ? 100 : 1));
</script>

<template>
    <FileUpload.Root
        v-model:accepted-files="model"
        :accept="accept"
        :max-files="maxFiles"
        class="flex flex-col gap-3"
    >
        <FileUpload.Dropzone
            class="border-border-1 hover:bg-surface-indent ease-out-expo flex flex-col items-center justify-center gap-1 rounded-xl border border-dashed px-4 py-8 text-center transition-colors"
        >
            <FileUpload.Trigger
                class="ds-focus-ring flex w-full cursor-pointer flex-col items-center gap-1 rounded-lg"
            >
                <Icon :name="icon" class="text-text-muted size-6" />
                <span v-if="label" class="text-body-3 text-text-default">
                    {{ label }}
                </span>
                <span v-if="description" class="text-caption-1 text-text-hint">
                    {{ description }}
                </span>
            </FileUpload.Trigger>
        </FileUpload.Dropzone>

        <FileUpload.ItemGroup class="flex flex-col gap-2">
            <FileUpload.Context v-slot="{ acceptedFiles }">
                <FileUpload.Item
                    v-for="(file, index) in acceptedFiles"
                    :key="`${file.name}:${index}`"
                    :file="file"
                    class="border-border-1 flex items-center gap-3 rounded-xl border px-3 py-2"
                >
                    <Icon
                        name="tabler:music"
                        class="text-text-muted size-5 shrink-0"
                    />
                    <div class="flex min-w-0 flex-col">
                        <FileUpload.ItemName
                            class="text-body-3 text-text-default truncate"
                        />
                        <FileUpload.ItemSizeText
                            class="text-caption-1 text-text-hint"
                        />
                    </div>
                    <slot name="file-trailing" :file="file" :index="index" />
                    <FileUpload.ItemDeleteTrigger
                        class="text-text-hint hover:text-semantic-error ds-focus-ring ml-auto shrink-0 cursor-pointer rounded-lg p-1"
                    >
                        <Icon name="tabler:x" class="size-4" />
                    </FileUpload.ItemDeleteTrigger>
                </FileUpload.Item>
            </FileUpload.Context>
        </FileUpload.ItemGroup>

        <FileUpload.HiddenInput />
    </FileUpload.Root>
</template>
