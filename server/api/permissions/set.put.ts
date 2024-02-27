import { Permissions } from "~/utils/permissions";

export default defineEventHandler(async (event) => {
    const email = getHeader(event, "x-token-user-email");

    if (!email) {
        setResponseStatus(event, 401);
        return;
    }

    const perms = await getPermissions(email);

    if (!perms?.admin) {
        setResponseStatus(event, 403);
        return;
    }

    const request = await readBody<{
        email: string;
        permissions: Permissions;
    }>(event);

    await setPermissions(request.email, request.permissions);
});
