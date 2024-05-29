<script setup lang="ts">
// theme
import VueForm from "@lljj/vue3-form-naive";
import type { SelectBaseOption } from "naive-ui/es/select/src/interface";
import { lightTheme } from "naive-ui";
import localizeEn from "ajv-i18n/localize/en";
import { i18n } from "@lljj/vue3-form-naive";
//import { enUS, dateEnUS } from "naive-ui";

const runtimeConfig = useRuntimeConfig();

type Workflow = {
    name: string;
    schema: any;
};
const { data, error, status } = useFetch<Workflow[]>(
    `${runtimeConfig.public.temporalTriggerUrl}/schemas`,
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
        return workflow?.schema;
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

const onSubmit = async (data: any) => {
    console.log(data);

    const response = await fetch(
        `http://${runtimeConfig.public.temporalTriggerUrl}/trigger-dynamic?workflow=${selectedWorkflow.value}`,
        {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(data.value),
        },
    );

    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }

    return await response.json();
};

i18n.useLocal(localizeEn);
</script>

<template>
    <div class="h-dvh w-full bg-emerald-400 p-4">
        <n-config-provider :theme="lightTheme">
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
                    filterable
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
                    @submit="onSubmit"
                />
                <n-collapse>
                    <n-collapse-item title="Schema">
                        <pre>{{ schema }}</pre>
                    </n-collapse-item>
                </n-collapse>
            </n-space>
        </n-config-provider>
    </div>
</template>

<style>
body {
    background: black;
}
</style>
