export function useDeleteMode() {
    return {
        deleteMode: useState("delete-mode", () => false),
    };
}
