<script lang="ts" setup>
import type { BmmEnvironment } from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    languages: string[];
    env: BmmEnvironment;
    label?: string;
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
            ).Languages.map((l) => l.code);
        }

        if (bmmLanguages.value.length == 1) {
            model.value = bmmLanguages.value[0];
        }
    },
    { immediate: true },
);

const languageDisplay = (l: string) => {
    if (typeof Intl.DisplayNames !== "undefined") {
        const dn = new Intl.DisplayNames(["en"], { type: "language" });
        let name = dn.of(l);

        // Chrome doesn't support "kha"
        if (name == "kha") {
            return "Khasi";
        } else if (name == "zxx") {
            return "Instrumental";
        }

        return dn.of(l);
    }
};

const items = computed(() => {
    const langs = [];

    langs.push(
        ...bmmLanguages.value.map((l) => ({
            label: languageDisplay(l),
            value: l,
        })),
    );

    return langs;
});
</script>

<template>
    <UFormField required :label="label ?? $t('Language')">
        <USelect
            v-model="model"
            :placeholder="$t('selectAnOption')"
            :items="
                bmmLanguages.map((l) => ({
                    label: languageDisplay(l),
                    value: l,
                }))
            "
            size="lg"
            class="w-full"
        />
    </UFormField>
</template>
