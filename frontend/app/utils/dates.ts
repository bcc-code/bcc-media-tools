import type { Timestamp } from "@bufbuild/protobuf/wkt";

export function timestampToDate(ts: Timestamp | undefined) {
	if (!ts) return;
	return new Date(Number(ts.seconds) * 1000 + Math.round(ts.nanos / 1e6));
}