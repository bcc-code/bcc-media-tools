/**
 * The app's phone breakpoint, as a reactive boolean.
 *
 * Must stay in sync with Tailwind's `md` breakpoint (768px) — i.e. the `max-md:`
 * variant — so CSS and JS flip at the same pixel. This is the same split the
 * header's hamburger/inline nav uses in `layouts/default.vue`.
 */
export const useIsPhone = () => useMediaQuery("(max-width: 767px)");
