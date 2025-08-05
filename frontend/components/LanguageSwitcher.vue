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
</script>

<template>
    <select
        v-model="localeComp"
        class="rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-sm shadow-sm dark:border-neutral-700 dark:bg-neutral-800"
    >
        <option
            v-for="locale in locales"
            :key="locale.code"
            :value="locale.code"
        >
            {{ locale.name }}
        </option>
    </select>
</template>
