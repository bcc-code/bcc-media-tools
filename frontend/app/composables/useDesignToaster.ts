import { createToaster } from "@ark-ui/vue";

/*
 * Renamed from admin-web's `useToast` to avoid clashing with Nuxt UI's own
 * `useToast`, which 1+ existing files still use while both systems coexist.
 * Pair with <DesignToastProvider />, mounted once in app.vue.
 */
export const useDesignToaster = () => {
    // useState memoizes the toaster across the app; unwrap to the instance so
    // callers can use `toaster.create(...)` directly (it's a stable object).
    return useState("design-toaster", () =>
        createToaster({
            placement: "bottom-end",
            overlap: false,
            gap: 12,
            duration: 5000,
        }),
    ).value;
};
