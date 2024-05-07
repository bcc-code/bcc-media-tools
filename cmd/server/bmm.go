package main

import (
	apiv1 "bcc-media-tools/gen/api/v1"
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"strconv"
	"time"
)

type BMMApi struct {
	client *resty.Client
	token  *BMMToken
}

func getToken(tokenBaseURL, clientID, clientSecret, audience string) (*BMMToken, error) {
	r := resty.New()
	r.BaseURL = tokenBaseURL
	res, err := r.R().SetBody(map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"audience":      audience,
		"grant_type":    "client_credentials",
	}).SetResult(&BMMToken{}).Post("/oauth/token")

	if err != nil {
		return nil, err
	}

	token := res.Result().(*BMMToken)
	token.CreatedAt = time.Now()

	return token, nil
}

func NewBMMToken(tokenBaseURL, clientID, clientSecret, audience string) (*BMMToken, error) {
	t, err := getToken(tokenBaseURL, clientID, clientSecret, audience)
	if err != nil {
		return nil, err
	}

	t.tokenBaseURL = tokenBaseURL
	t.clientID = clientID
	t.clientSecret = clientSecret
	t.audience = audience

	return t, nil
}

type BMMToken struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	CreatedAt   time.Time

	tokenBaseURL string
	clientID     string
	clientSecret string
	audience     string
}

func (t *BMMToken) GetAccessToken() string {
	if t.Expired() {
		err := t.Refresh()
		if err != nil {
			// TODO: Maybe not panic?
			panic(err)
		}
	}

	return t.AccessToken
}

func debugResponse(resp *resty.Response) {
	// Explore response object
	fmt.Println("Response Info:")
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Proto      :", resp.Proto())
	fmt.Println("  Time       :", resp.Time())
	fmt.Println("  Received At:", resp.ReceivedAt())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()

	// Explore trace info
	fmt.Println("Request Trace Info:")
	ti := resp.Request.TraceInfo()
	fmt.Println("  DNSLookup     :", ti.DNSLookup)
	fmt.Println("  ConnTime      :", ti.ConnTime)
	fmt.Println("  TCPConnTime   :", ti.TCPConnTime)
	fmt.Println("  TLSHandshake  :", ti.TLSHandshake)
	fmt.Println("  ServerTime    :", ti.ServerTime)
	fmt.Println("  ResponseTime  :", ti.ResponseTime)
	fmt.Println("  TotalTime     :", ti.TotalTime)
	fmt.Println("  IsConnReused  :", ti.IsConnReused)
	fmt.Println("  IsConnWasIdle :", ti.IsConnWasIdle)
	fmt.Println("  ConnIdleTime  :", ti.ConnIdleTime)
	fmt.Println("  RequestAttempt:", ti.RequestAttempt)
	fmt.Println("  RemoteAddr    :", ti.RemoteAddr.String())
}

func (t *BMMToken) Expired() bool {
	return time.Since(t.CreatedAt)+10*time.Second > time.Duration(t.ExpiresIn)
}

func (t *BMMToken) Refresh() error {
	newToken, err := getToken(t.tokenBaseURL, t.clientID, t.clientSecret, t.audience)
	if err != nil {
		return err
	}

	t.AccessToken = newToken.AccessToken
	t.Scope = newToken.Scope
	t.ExpiresIn = newToken.ExpiresIn
	t.TokenType = newToken.TokenType
	t.CreatedAt = time.Now()

	return nil
}

func NewBMMApi(baseURL string, token *BMMToken) *BMMApi {
	b := &BMMApi{}
	b.client = resty.New()
	b.client.BaseURL = baseURL
	b.client.SetHeader("Accept-Language", "en")
	b.token = token

	return b
}

type bmmYear struct {
	Year  uint32 `json:"year"`
	Count uint32 `json:"count"`
}

func (a BMMApi) GetYears(_ context.Context, req *connect.Request[apiv1.Void]) (*connect.Response[apiv1.GetYearsResponse], error) {
	yearReq := a.client.R().
		SetAuthToken(a.token.GetAccessToken()).
		SetResult(&[]bmmYear{})

	yearsResponse, err := yearReq.Get("/facets/album_published/years")
	if err != nil {
		return nil, err
	}

	out := &apiv1.GetYearsResponse{
		Data: make(map[uint32]*apiv1.BMMYear),
	}
	for _, y := range *yearsResponse.Result().(*[]bmmYear) {
		out.Data[y.Year] = &apiv1.BMMYear{
			Year:  y.Year,
			Count: y.Count,
		}
	}

	return connect.NewResponse(out), nil
}

type BMMAlbum struct {
	Meta      Meta        `json:"_meta"`
	BmmID     interface{} `json:"bmm_id"`
	Cover     string      `json:"cover"`
	ID        int         `json:"id"`
	Languages []string    `json:"languages"`
	//ParentID    interface{} `json:"parent_id"`
	PublishedAt time.Time `json:"published_at"`
	Tags        []string  `json:"tags"`
	Language    string    `json:"language"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
}

type Meta struct {
	ContainedTypes []string  `json:"contained_types"`
	IsVisible      bool      `json:"is_visible"`
	ModifiedAt     time.Time `json:"modified_at"`
	ModifiedBy     string    `json:"modified_by"`
}

func (a BMMApi) GetAlbums(_ context.Context, req *connect.Request[apiv1.GetAlbumsRequest]) (*connect.Response[apiv1.AlbumsList], error) {
	albumsReq := a.client.R().
		SetAuthToken(a.token.GetAccessToken()).
		SetResult(&[]BMMAlbum{})

	albumsResponse, err := albumsReq.Get(fmt.Sprintf("/album/published/%d/", req.Msg.Year))
	if err != nil {
		return nil, err
	}

	albums := albumsResponse.Result().(*[]BMMAlbum)
	out := &apiv1.AlbumsList{
		Albums: make([]*apiv1.Album, len(*albums)),
	}

	for i, a := range *albums {
		out.Albums[i] = &apiv1.Album{
			Id:        strconv.Itoa(a.ID),
			Title:     a.Title,
			Languages: a.Languages,
			Cover:     a.Cover,
		}
	}

	return connect.NewResponse(out), nil
}
