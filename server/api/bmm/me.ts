export default defineEventHandler(async (event) => {
    const email = getHeader(event, "x-token-user-email");
    if (!email) {
        setResponseStatus(event, 401);
        return;
    }

    return await getPermissions(email);
});
