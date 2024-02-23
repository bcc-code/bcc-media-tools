<script lang="ts" setup>
import { BccButton, BccInput } from "@bcc-code/design-library-vue";

const form = defineModel<BMMSingleForm>({ required: true });

const language = computedProperty(form, "language");
const originalTitle = computedProperty(form, "originalTitle");

defineEmits<{
    submit: [];
}>();
</script>
<template>
    <form class="flex flex-col gap-4 p-4" @submit.prevent="$emit('submit')">
        <h3 class="text-lg font-bold">BMM Upload</h3>
        <BccInput
            v-model="originalTitle"
            :label="$t('originalTitle')"
            required
        />
        <LanguageSelector v-model="language" :languages="bmmLanguages" />
        <BccButton type="submit">Next</BccButton>
    </form>
</template>
