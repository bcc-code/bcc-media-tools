/**
 * Builds the "Go to job" action for toasts fired after starting a workflow, so
 * users can jump straight to its status on the jobs dashboard.
 */
export function useJobsToast() {
    const { t } = useI18n();

    // Pass a single asset's VXID to deep-link to its jobs (filtered by
    // reference); omit it to open the unfiltered, newest-first list.
    function goToJobsAction(reference?: string) {
        return {
            label: t("jobs.goToJob"),
            onClick: () =>
                navigateTo(
                    reference
                        ? `/jobs?ref=${encodeURIComponent(reference)}`
                        : "/jobs",
                ),
        };
    }

    return { goToJobsAction };
}
