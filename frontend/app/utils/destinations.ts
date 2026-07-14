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
    ["hippo", "Hippo"],
    ["hippo_v2", "Hippo v2"],
    ["hippo_hap", "Hippo HAP"],
    ["dubbing", "Dubbing"],
    ["hyperdeck", "Hyperdeck"],
    ["caspar-cg", "Caspar CG"],
]);

export function destinationName(value: string): string {
    return DestinationNames.get(value) ?? value;
}

// destinationOptions maps destination values to { label, value } items for
// USelect multi-selects. The list of values is supplied by the backend
// (GetExportDestinations) so the admin editor can never offer a destination the
// backend doesn't accept.
export function destinationOptions(values: string[]) {
    return values.map((value) => ({ label: destinationName(value), value }));
}
