export default defineEventHandler(async (event) => {
    const album = getRouterParam(event, "album");

    switch (album) {
        case "fra-kaare":
            return await getFraKaareTracks();
    }
    return [];
});
