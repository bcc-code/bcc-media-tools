// Allowlist of subtitle burn-in styles offered in the VB Export tool.
//
// The backend lists every file in SUBTITLE_STYLES_DIR, so old and experimental
// styles on the share show up in the picker. Edit the list below to control
// what is actually selectable — `name` is the style name WITHOUT its file
// extension, matched case-insensitively.
//
// The picker shows the styles in the order they are listed here.
//
// `key` points at `vbExport.subtitleStyleInfo.<key>` in the locale files, which
// holds a human-readable `title` and `description` for the style. Leave it out
// to show the bare style name with no explainer.
//
// An empty list disables the allowlist and offers everything the backend sends.
type SubtitleStyleEntry = {
    name: string;
    key?: string;
};

const AllowedSubtitleStyles: SubtitleStyleEntry[] = [
    { name: "03-brunstad-to-linjer", key: "twoLines" },
    { name: "03-brunstad-led-pc25", key: "led" },
    { name: "03-brunstad-to-linjer_v2", key: "twoLinesV2" },
];

const allowed = new Map(
    AllowedSubtitleStyles.map((s) => [s.name.toLowerCase(), s]),
);

// subtitleStyleLabel strips the file extension off a style filename, e.g.
// "03-brunstad-led-pc25.ass" -> "03-brunstad-led-pc25".
export function subtitleStyleLabel(style: string): string {
    return style.replace(/\.[^./]+$/, "");
}

// subtitleStyleInfoKey returns the i18n key suffix for a style filename, or
// undefined when the style has no explainer in the locale files.
export function subtitleStyleInfoKey(style: string): string | undefined {
    return allowed.get(subtitleStyleLabel(style).toLowerCase())?.key;
}

// allowedSubtitleStyles filters the backend's style filenames down to the ones
// listed above, in the order they are listed.
export function allowedSubtitleStyles(styles: string[]): string[] {
    if (allowed.size === 0) return styles;
    const byLabel = new Map(
        styles.map((s) => [subtitleStyleLabel(s).toLowerCase(), s]),
    );
    return AllowedSubtitleStyles.map((e) =>
        byLabel.get(e.name.toLowerCase()),
    ).filter((s) => s !== undefined);
}
