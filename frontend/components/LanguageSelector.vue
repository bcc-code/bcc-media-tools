<script lang="ts" setup>
import { BccSelect } from "@bcc-code/design-library-vue";
import type { BmmEnvironment } from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    languages: string[];
    env: BmmEnvironment;
}>();

const model = defineModel<string>();
const bmmLanguages = ref<string[]>([]);

const api = useAPI();

watch(
    [() => props.env, () => props.languages],
    async ([newEnv, newLanguages]) => {
        if (newLanguages.length > 0) {
            bmmLanguages.value = newLanguages;
        } else {
            bmmLanguages.value = (
                await api.getLanguages({ environment: newEnv })
            ).Languages;
        }
    },
    { immediate: true },
);

const languageDisplay = (l: string) => {
    if (typeof Intl.DisplayNames !== "undefined") {
        const dn = new Intl.DisplayNames(["en"], { type: "language" });
        return dn.of(l);
    }
};
</script>

<template>
    <BccSelect  :required="true" v-model="model" :label="$t('language')">
        <option disabled value="">{{ $t("selectAnOption") }}</option>
        <option v-for="l in bmmLanguages" :value="l">
            {{ languageDisplay(l) }}
        </option>
    </BccSelect>
</template>
