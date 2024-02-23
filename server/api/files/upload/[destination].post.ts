import { IncomingForm } from "formidable";

export default defineEventHandler((event) => {
    const destination = getRouterParam(event, "destination");
    if (!destination) {
        return;
    }
    if (destination !== "bmm") {
        return;
    }

    const tempDrive = useRuntimeConfig().api.tempDrivePath;
    const uploadDir = tempDrive + "/tools/uploads/" + destination;

    const form = new IncomingForm({
        uploadDir,
        filename(name, _ext, part, _form) {
            return part.originalFilename ?? name;
        },
        createDirsFromUploads: true,
    });

    return new Promise((resolve, reject) => {
        form.parse(event.node.req, (err, fields, files) => {
            if (err) {
                reject(err);
                return;
            }
            resolve({ fields, files });
        });
    });
});
