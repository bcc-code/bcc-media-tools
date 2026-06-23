<script setup>
const { locale, locales, setLocale } = useI18n();

const localeComp = computed({
    get() {
        return locale.value;
    },
    set(v) {
        setLocale(v);
    },
});

const items = computed(() =>
    locales.value.map((locale) => ({
        label: locale.name,
        value: locale.code,
    })),
);
</script>

<template>
    <DesignSelect v-model="localeComp" :items="items" />
</template>
