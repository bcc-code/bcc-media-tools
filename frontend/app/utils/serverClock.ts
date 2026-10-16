// Measuring how far this computer's clock is from the server's, NTP-style:
// note when a request left and when its answer came back, and assume the
// server read its clock halfway through. The quickest round trip is the least
// distorted by network delay, so that sample wins.

// One round trip: sentAt and receivedAt on this computer's clock, serverMs
// the server's.
export interface ClockSample {
    sentAt: number;
    receivedAt: number;
    serverMs: number;
}

export interface ClockOffset {
    // add to this computer's time to get the server's
    offsetMs: number;
    // how wrong offsetMs can be: half the round trip
    uncertaintyMs: number;
}

export function bestOffset(samples: ClockSample[]): ClockOffset | undefined {
    let best: ClockSample | undefined;
    for (const s of samples) {
        if (s.receivedAt < s.sentAt) continue;
        if (!best || s.receivedAt - s.sentAt < best.receivedAt - best.sentAt)
            best = s;
    }
    if (!best) return undefined;
    return {
        offsetMs: best.serverMs - (best.sentAt + best.receivedAt) / 2,
        uncertaintyMs: (best.receivedAt - best.sentAt) / 2,
    };
}

// A positive offset means the server is ahead, so this computer is behind.
export function describeOffset(offsetMs: number): {
    ahead: boolean;
    seconds: string;
} {
    return {
        ahead: offsetMs < 0,
        seconds: (Math.abs(offsetMs) / 1000).toFixed(1),
    };
}
