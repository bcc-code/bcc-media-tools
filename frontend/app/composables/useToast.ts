import { createToaster } from "@ark-ui/vue";

export const useToast = () => {
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
