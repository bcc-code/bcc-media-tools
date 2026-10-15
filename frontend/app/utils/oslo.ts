// Wall-clock helpers for Europe/Oslo, the zone editors read times in. Date
// and time inputs are Oslo wall-clock; the API is UTC.

// Europe/Oslo's offset at an instant: format it there, read it back as UTC,
// and the gap is the offset. The inputs are wall-clock; the API is UTC.
export function osloOffsetMs(at: Date): number {
    const parts = new Intl.DateTimeFormat("en-GB", {
        timeZone: "Europe/Oslo",
        hourCycle: "h23",
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
    }).formatToParts(at);
    const n = (type: string) =>
        Number(parts.find((p) => p.type === type)?.value ?? 0);
    return (
        Date.UTC(
            n("year"),
            n("month") - 1,
            n("day"),
            n("hour"),
            n("minute"),
            n("second"),
        ) - at.getTime()
    );
}

// Reads "YYYY-MM-DD" + "HH:MM" as Oslo wall-clock time. Resolved twice: the
// first pass can land on the wrong side of a DST change.
export function osloToDate(date: string, time: string): Date | undefined {
    const d = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date.trim());
    const t = /^(\d{1,2}):(\d{2})$/.exec(time.trim());
    if (!d || !t || !isClockTime(time)) return undefined;
    const naive = Date.UTC(+d[1]!, +d[2]! - 1, +d[3]!, +t[1]!, +t[2]!);
    const first = new Date(naive - osloOffsetMs(new Date(naive)));
    return new Date(naive - osloOffsetMs(first));
}

// The "YYYY-MM-DD" and "HH:MM" an Oslo date and time input show for an instant.
export function osloDateInput(at: Date): string {
    return new Intl.DateTimeFormat("en-CA", { timeZone: "Europe/Oslo" }).format(
        at,
    );
}

export function osloTimeInput(at: Date): string {
    return new Intl.DateTimeFormat("en-GB", {
        timeZone: "Europe/Oslo",
        timeStyle: "short",
    }).format(at);
}

// A 24-hour "HH:MM" (or "H:MM"). Times are typed as text: a native time input
// follows the browser's locale and may show AM/PM.
export function isClockTime(text: string): boolean {
    const m = /^(\d{1,2}):(\d{2})$/.exec(text.trim());
    return !!m && +m[1]! < 24 && +m[2]! < 60;
}
