import { IncomingForm } from "formidable";

export default defineEventHandler((event) => {
    const form = new IncomingForm({
        uploadDir: "uploads",
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

            // Process the uploaded files as needed. For example, you could return the file path.
            // This is a simplistic example; adjust according to your needs.
            resolve({ fields, files });
        });
    });
});
