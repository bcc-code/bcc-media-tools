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
        class="border-accented bg-default rounded-lg border px-3 py-1.5 text-sm shadow-sm"
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
