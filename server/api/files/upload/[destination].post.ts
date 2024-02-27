import { Fields, File, IncomingForm } from "formidable";
import { v4 } from "uuid";

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

    const destination = getRouterParam(event, "destination");
    if (!destination) {
        return;
    }
    if (destination !== "bmm") {
        return;
    }

    const tempDrive = useRuntimeConfig().api.tempDrivePath;
    const now = new Date();
    const uploadDir =
        tempDrive +
        "/tools/uploads/" +
        destination +
        "/" +
        `${now.getFullYear()}-${now.getMonth() + 1}-${now.getDate()}/` +
        v4().substring(0, 8);

    const form = new IncomingForm({
        uploadDir,
        filename(name, _ext, part, _form) {
            return part.originalFilename ?? name;
        },
        createDirsFromUploads: true,
    });

    const { fields, files } = await new Promise<{
        fields: Fields<string>;
        files: File[] | null;
    }>((resolve, reject) => {
        form.parse(event.node.req, (err, fields, files) => {
            if (err) {
                reject(err);
                return;
            }
            resolve({
                fields,
                files: files.file ?? null,
            });
        });
    });

    if (!files || files.length !== 1) {
        return;
    }

    const title = fields.title?.[0];
    const trackId = parseInt(fields.trackId?.[0] ?? "0");
    const language = fields.language?.[0];

    await fetch(
        useRuntimeConfig().api.temporalTriggerUrl +
            "/trigger/WebHook?type=bmm_simple_upload",
        {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                title,
                trackId,
                language,
                filePath: files[0].filepath,
                uploadedBy: email,
            }),
        },
    );
});
