// A resolved asset shown in the bulk-export list: a VX-id and its title.
export interface AssetRef {
    vxId: string;
    title: string;
    // false when the backend could not resolve the id (unknown / inaccessible)
    found?: boolean;
}

// Matches Vidispine item ids, e.g. "VX-123456". Mirrors the validation regex
// used elsewhere (/^VX-[a-zA-Z0-9]+$/).
const VXID_RE = /VX-[A-Za-z0-9]+/g;

// extractVXIDs pulls every VX-id out of arbitrary pasted text, deduplicated and
// in first-seen order.
export function extractVXIDs(text: string): string[] {
    return [...new Set(text.match(VXID_RE) ?? [])];
}
