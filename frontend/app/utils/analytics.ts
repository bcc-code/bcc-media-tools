import type { RudderAnalytics } from "@rudderstack/analytics-js";
import { useAPI } from "~/utils/api";

export class Analytics {
    private initialized = false;
    private user: IdentifyData | null = null;

    private pageQueue: Array<any> = [];
    private trackQueue: Array<any> = [];

    constructor(private rudder: RudderAnalytics) {}

    public getUser() {
        return this.user;
    }

    public setUser(user: IdentifyData) {
        this.initialized = true;
        const data = Object.assign({}, user) as any;
        this.user = data;
        delete data["id"];
        this.rudder.identify(user.Email, data);
    }

    public page(page: {
        id: Page;
        title: string;
        meta?: {
            setting?: "webSettings";
            [key: string]: string | undefined;
        };
    }) {
        if (!this.initialized) {
            this.pageQueue.push(page);
            return;
        }

        const data = Object.assign({}, page) as any;
        delete data["id"];
        this.rudder.page(page.id, data);
    }

    public track<T extends keyof Events>(event: T, data: Events[T]) {
        if (!this.initialized) {
            this.trackQueue.push({ event, data });
            return;
        }

        this.rudder.track(event, {
            ...data,
        });
    }

    public async initialize() {
        const api = useAPI();

        this.setUser({
            Email: (await api.getPermissions({})).email,
        });

        this.initialized = true;

        this.pageQueue.forEach((p) => this.page(p));
        this.pageQueue = [];

        this.trackQueue.forEach(({ event: e, data: d }, _, __) =>
            this.track(e, d),
        );
    }
}

interface IdentifyData {
    Email: string;
}
