export default function computedProperty<T, K extends keyof T>(
    mutable: Ref<T | undefined>,
    property: K,
): globalThis.Ref<T[K] | undefined> {
    return computed({
        get() {
            return mutable.value?.[property];
        },
        set(v) {
            mutable.value = {
                ...(mutable.value ?? ({} as T)),
                [property]: v,
            };
        },
    });
}
