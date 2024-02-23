export default <T extends {}>(ref: Ref<T | undefined>, property: keyof T) => {
    return computed({
        get() {
            return ref.value?.[property];
        },
        set(v) {
            ref.value = ref.value
                ? {
                      ...ref.value,
                      [property]: v,
                  }
                : undefined;
        },
    });
};
