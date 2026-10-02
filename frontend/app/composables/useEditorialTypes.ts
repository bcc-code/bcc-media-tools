import { EDITORIAL_TYPES } from "~/utils/editorial";

/** Translated marker-type labels, falling back to the raw (legacy) value. */
export function useEditorialTypes() {
    const { t, te } = useI18n();

    function typeLabel(type: string): string {
        const key = `editorial.types.${type}`;
        if (te(key)) return t(key);
        return type ? type.charAt(0).toUpperCase() + type.slice(1) : type;
    }

    const typeItems = computed(() =>
        EDITORIAL_TYPES.map((value) => ({ label: typeLabel(value), value })),
    );

    return { typeLabel, typeItems };
}
