import { describe, expect, it, beforeEach, vi } from "vitest";
import { computed } from "vue";
import { mount } from "@vue/test-utils";
import { mockNuxtImport } from "@nuxt/test-utils/runtime";
import DesignDialog from "~/components/design/DesignDialog.vue";
import DesignSheet from "~/components/design/DesignSheet.vue";

const viewport = vi.hoisted(() => ({ phone: false }));

mockNuxtImport("useIsPhone", () => () => computed(() => viewport.phone));

/** Both components teleport into the app's shared portal node. */
function portal() {
    let el = document.getElementById("teleports");
    if (!el) {
        el = document.createElement("div");
        el.id = "teleports";
        document.body.appendChild(el);
    }
    return el;
}

function part(scope: string, name: string) {
    return document.querySelector(
        `[data-scope="${scope}"][data-part="${name}"]`,
    );
}

describe("DesignDialog", () => {
    beforeEach(() => {
        portal().innerHTML = "";
        viewport.phone = false;
    });

    it("renders a centred dialog on desktop", async () => {
        mount(DesignDialog, {
            props: { open: true, title: "Export" },
            attachTo: document.body,
        });
        await nextTick();

        expect(part("dialog", "content")).not.toBeNull();
        expect(part("drawer", "content")).toBeNull();
    });

    it("renders a draggable sheet below md:", async () => {
        viewport.phone = true;
        mount(DesignDialog, {
            props: { open: true, title: "Export" },
            attachTo: document.body,
        });
        await nextTick();

        expect(part("drawer", "content")).not.toBeNull();
        // The grabber is what makes it swipe-dismissable.
        expect(part("drawer", "grabber")).not.toBeNull();
        expect(part("dialog", "content")).toBeNull();
    });

    it("keeps the title and description wired up in sheet form", async () => {
        viewport.phone = true;
        mount(DesignDialog, {
            props: { open: true, title: "Export", description: "Pick a file" },
            attachTo: document.body,
        });
        await nextTick();

        expect(part("drawer", "title")?.textContent).toContain("Export");
        expect(part("drawer", "description")?.textContent).toContain(
            "Pick a file",
        );
    });
});

describe("overlay enter/exit motion", () => {
    beforeEach(() => {
        portal().innerHTML = "";
        viewport.phone = false;
    });

    // Ark mounts through @zag-js/presence, which detects the exit by reading
    // `animationName` and waiting for `animationend`. A CSS transition is
    // invisible to it: the node is torn down on the spot and the overlay blinks
    // out. So the state classes must resolve to keyframe animations.
    it.each([
        ["sheet", true, "drawer"],
        ["dialog", false, "dialog"],
    ])(
        "drives %s exit with an animation, not a transition",
        async (_label, phone, scope) => {
            viewport.phone = phone;
            mount(DesignDialog, {
                props: { open: true, title: "Export" },
                attachTo: document.body,
            });
            await nextTick();

            for (const name of ["content", "backdrop"]) {
                const cls = part(scope, name)!.className;
                expect(cls).toMatch(/data-\[state=closed\]:[a-z-]*(exit)/);
                expect(cls).toMatch(/data-\[state=open\]:[a-z-]*(enter)/);
                expect(cls).not.toMatch(
                    /data-\[state=closed\]:(opacity|translate|scale)/,
                );
            }
        },
    );
});

describe("DesignSheet", () => {
    beforeEach(() => {
        portal().innerHTML = "";
    });

    it("renders drawer parts directly", async () => {
        mount(DesignSheet, {
            props: { open: true, title: "Filters" },
            attachTo: document.body,
        });
        await nextTick();

        expect(part("drawer", "content")).not.toBeNull();
        expect(part("drawer", "grabber")).not.toBeNull();
        expect(part("drawer", "backdrop")).not.toBeNull();
    });
});
