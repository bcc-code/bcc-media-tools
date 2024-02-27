export default defineEventHandler(async (event) => {
    const email = getHeader(event, "x-token-user-email");

    if (!email) {
        setResponseStatus(event, 401);
        return;
    }

    const perms = await getPermissions(email);

    const album = getRouterParam(event, "album");

    switch (album) {
        case "fra-kaare":
            if (!perms?.bmm.albums.includes("fra-kaare")) {
                setResponseStatus(event, 403);
                return;
            }
            return await getFraKaareTracks();
    }
    return [];
});
