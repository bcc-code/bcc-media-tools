import { readFile, writeFile } from "fs/promises";

let allPermissions: {
    [key: string]: Permissions | undefined;
};

const getAllPermissions = async () => {
    return (allPermissions ??= JSON.parse(
        await readFile("./config/permissions.json", { encoding: "utf-8" }),
    ));
};

export type Permissions = {
    admin: boolean;
    bmm: {
        languages: string[];
        albums: string[];
    };
};

export async function getPermissions(email: string) {
    const perms = await getAllPermissions();

    return perms[email] ?? null;
}

export async function setPermissions(
    email: string,
    permissions: Permissions | null,
) {
    const perms = await getAllPermissions();

    if (permissions) {
        perms[email] = permissions;
    } else {
        delete perms[email];
    }

    await writeFile(
        "./config/permissions.json",
        JSON.stringify(perms, null, 4),
    );
}

export async function listPermissions() {
    return await getAllPermissions();
}
