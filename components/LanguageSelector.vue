<script lang="ts" setup>
import { BccSelect } from "@bcc-code/design-library-vue";

defineProps<{
    languages: readonly string[];
}>();

const value = defineModel<string>();

const languageDisplay = (l: string) => {
    if (typeof Intl.DisplayNames !== "undefined") {
        const dn = new Intl.DisplayNames(["en"], { type: "language" });
        return dn.of(l);
    }
};
</script>

<template>
    <BccSelect v-model="value" :label="$t('language')">
        <option disabled value="">{{ $t("selectAnOption") }}</option>
        <option v-for="l in languages" :value="l">
            {{ languageDisplay(l) }}
        </option>
    </BccSelect>
</template>
