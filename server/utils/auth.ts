export class Auth{
    private _tokenEndpoint;
    private _audience;
    private _clientId;
    private _clientSecret;

    constructor(tokenEndpoint: string, audience: string, clientId: string, clientSecret: string) {
        this._tokenEndpoint = tokenEndpoint;
        this._audience = audience;
        this._clientId = clientId;
        this._clientSecret = clientSecret;
    }

    private _expiry: Date | null = null;
    private _token: string | null = null;
    
    public async getToken() {
        if (this._token && this._expiry && this._expiry > new Date()) {
            return this._token;
        }

        const response = await fetch(this._tokenEndpoint, {
            method: "POST",
            headers: {
                "Content-Type": "application/x-www-form-urlencoded",
                "Accept": "application/json"
            },
            body: new URLSearchParams({
                grant_type: "client_credentials",
                client_id: this._clientId,
                client_secret: this._clientSecret,
                audience: this._audience,
            }),
        });

        if (!response.ok) {
            throw new Error("Failed to fetch token");
        }

        const {access_token, token_type, expires_in} = await response.json();

        this._expiry = new Date(Date.now() + expires_in * 1000);

        return this._token = `${token_type} ${access_token}`;
    }
}