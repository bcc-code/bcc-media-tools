import { identify, page as rpage, track } from "rudder-sdk-js";
import { useAPI } from "~/utils/api";

export class Analytics {
    private initialized = false;
    private user: IdentifyData | null = null;

    private pageQueue: Array<any> = [];
    private trackQueue: Array<any> = [];

    public getUser() {
        return this.user;
    }

    public setUser(user: IdentifyData) {
        this.initialized = true;
        const data = Object.assign({}, user) as any;
        this.user = data;
        delete data["id"];
        identify(user.Email, data);
    }

    public page(page: {
        id: Page;
        title: string;
        meta?: {
            setting?: "webSettings";
        };
    }) {
        if (!this.initialized) {
            this.pageQueue.push(page)
            return
        }

        const data = Object.assign({}, page) as any;
        delete data["id"];
        rpage(page.id, data);
    }

    public track<T extends keyof Events>(event: T, data: Events[T]) {
        if (!this.initialized) {
            this.trackQueue.push({ event, data })
            return
        };

        track(
            event,
            {
                ...data,
            },
            undefined,
            undefined,
        );
    }

    public async initialize() {
        const api = useAPI()

        this.setUser({
            Email: (await api.getPermissions({})).email
        });

        this.initialized = true;

        this.pageQueue.forEach(p => this.page(p))
        this.pageQueue = []

        this.trackQueue.forEach(({ event: e, data: d }, _, __) => this.track(e, d))
    }
}

interface IdentifyData {
    Email: string;
}

