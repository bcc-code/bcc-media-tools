// Friendly, human-readable names for export destinations.
//
// Edit the labels on the right to change how destinations appear in the UI
// (forms and the admin permission editor). The keys on the left are the
// technical destination values used by the backend/workflows and MUST NOT be
// changed. A value without an entry here falls back to showing the raw value.

const DestinationNames = new Map<string, string>([
    // VX export destinations
    ["vod", "VOD"],
    ["xdcam", "XDCAM"],
    ["bmm", "BMM"],
    ["bmm-integration", "BMM Integration"],
    ["isilon", "Isilon"],
    // VB export destinations
    ["abekas", "Abekas"],
    ["raw-abekas", "Raw Abekas"],
    ["b-stage", "B-Stage"],
    ["gfx", "GFX"],
    ["hippo_v2", "Hippo v2"],
    ["hippo_hap", "Hippo HAP"],
    ["dubbing", "Dubbing"],
    ["hyperdeck", "Hyperdeck"],
    ["caspar-cg", "Caspar CG"],
]);

export function destinationName(value: string): string {
    return DestinationNames.get(value) ?? value;
}

// Canonical full destination lists, used by the admin permission editor (which
// must show every selectable destination, not just the ones the user already
// has). Keep these in sync with the backend enums in bcc-media-flows.
export const VX_EXPORT_DESTINATIONS = [
    "xdcam",
    "vod",
    "bmm",
    "bmm-integration",
    "isilon",
];

export const VB_EXPORT_DESTINATIONS = [
    "abekas",
    "raw-abekas",
    "b-stage",
    "gfx",
    "hippo_v2",
    "hippo_hap",
    "dubbing",
    "hyperdeck",
    "xdcam",
    "caspar-cg",
];

// destinationOptions maps destination values to { label, value } items for
// USelect multi-selects.
export function destinationOptions(values: string[]) {
    return values.map((value) => ({ label: destinationName(value), value }));
}
