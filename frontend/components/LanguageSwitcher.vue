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
        class="rounded-lg border border-gray-300 bg-white px-4 py-2.5 shadow-sm"
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
