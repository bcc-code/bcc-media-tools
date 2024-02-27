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

    return await listPermissions();
});
