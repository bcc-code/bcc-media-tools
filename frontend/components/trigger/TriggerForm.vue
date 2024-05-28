<script setup lang="ts">
// theme
import VueForm from "@lljj/vue3-form-naive";
import type { SelectBaseOption } from "naive-ui/es/select/src/interface";
import { darkTheme } from "naive-ui";
import localizeEn from "ajv-i18n/localize/en";
import { i18n } from "@lljj/vue3-form-naive";
// locale & dateLocale
//import { enUS, dateEnUS } from "naive-ui";

type Workflow = {
    name: string;
    schema: any;
};
// fetch http://localhost:8080/schemas to get jsonschema with useFetch
const { data, error, status } = useFetch<Workflow[]>(
    "http://localhost:8080/schemas",
);

const workflows = computed(() => {
    return data.value?.map((workflow) => workflow);
});

const selectedWorkflow = ref<string | null>(null);

const schema = computed(() => {
    if (selectedWorkflow.value) {
        const workflow = workflows.value?.find(
            (workflow) => workflow.name === selectedWorkflow.value,
        );
        const o = workflow?.schema.$defs;
        return Object.values(o)[0];
    }
    return null;
});

const formData = ref({});

const options = computed(() => {
    return workflows.value?.map<SelectBaseOption>((workflow) => {
        return {
            label: workflow.name,
            value: workflow.name,
        };
    });
});

i18n.useLocal(localizeEn);
</script>

<template>
    <n-config-provider :theme="darkTheme">
        <n-space vertical>
            <!-- loading spinner if lading-->
            <n-spin v-if="status.value === 'pending'" />
            <!-- error message if error-->
            <n-alert type="error" v-if="error">{{ error }}</n-alert>
        </n-space>
        <n-space vertical v-if="data">
            <!-- select workflow -->
            <n-select
                v-model:value="selectedWorkflow"
                clearable
                :options="options"
            />
            {{ selectedWorkflow }}
            <VueForm
                v-if="schema"
                v-model="formData"
                :schema="schema"
                :uiSchema="schema"
                :formFooter="{
                    okBtn: 'Save',
                    cancelBtn: 'Cancel',
                }"
            />
            <pre>{{ schema }}</pre>
        </n-space>
    </n-config-provider>
</template>

<style>
body {
    background: black;
}
</style>
