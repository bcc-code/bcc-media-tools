// Allowlist of subtitle burn-in styles offered in the VB Export tool.
//
// The backend lists every file in SUBTITLE_STYLES_DIR, so old and experimental
// styles on the share show up in the picker. Edit the list below to control
// what is actually selectable — entries are the style name WITHOUT its file
// extension (the label shown on the card), matched case-insensitively.
//
// An empty list disables the allowlist and offers everything the backend sends.
const AllowedSubtitleStyles = [
    "03-brunstad-led-pc25",
    "03-brunstad-to-linjer",
    "03-brunstad-to-linjer_v2",
];

const allowed = new Set(AllowedSubtitleStyles.map((s) => s.toLowerCase()));

// subtitleStyleLabel strips the file extension off a style filename, e.g.
// "03-brunstad-led-pc25.ass" -> "03-brunstad-led-pc25".
export function subtitleStyleLabel(style: string): string {
    return style.replace(/\.[^./]+$/, "");
}

// allowedSubtitleStyles filters the backend's style filenames down to the ones
// listed above, preserving the backend's (alphabetical) order.
export function allowedSubtitleStyles(styles: string[]): string[] {
    if (allowed.size === 0) return styles;
    return styles.filter((s) =>
        allowed.has(subtitleStyleLabel(s).toLowerCase()),
    );
}
