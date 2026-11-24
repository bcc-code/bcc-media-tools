// Under 26 at the recording, for a suggestion; the server sends age 0 when unknown.
export function isU26(age: number | undefined): boolean {
    return !!age && age < 26;
}

// "Church · 34": what tells two people of the same name apart.
export function memberDetails(p: { age?: number; church?: string }): string {
    return [p.church, p.age || ""].filter(Boolean).join(" · ");
}
