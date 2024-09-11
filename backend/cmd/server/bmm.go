package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"github.com/samber/lo"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewBMMApi(baseURL string, token *BMMToken) *BMMApi {
	b := &BMMApi{}
	b.client = resty.New()
	b.client.BaseURL = baseURL
	b.client.SetHeader("Accept-Language", "nb")
	b.token = token

	return b
}

type bmmYear struct {
	Year  uint32 `json:"year"`
	Count uint32 `json:"count"`
}

func (a BMMApi) GetYears(_ context.Context, req *connect.Request[apiv1.GetYearsRequest]) (*connect.Response[apiv1.GetYearsResponse], error) {
	if !PermissionsForEmail(getEmail(req)).CanUpload() {
		return nil, connect.NewError(403, fmt.Errorf("not authorized"))
	}

	if req.Msg.Environment == apiv1.BmmEnvironment_Integration {
		a.client.BaseURL = os.Getenv("BMM_INT_BASE_URL")
	} else {
		a.client.BaseURL = os.Getenv("BMM_BASE_URL")
	}

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

type Meta struct {
	ContainedTypes []string  `json:"contained_types"`
	IsVisible      bool      `json:"is_visible"`
	ModifiedAt     time.Time `json:"modified_at"`
	ModifiedBy     string    `json:"modified_by"`
}

type BMMItem struct {
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
	Tracks      []BMMItem `json:"children"`
}
type BMMApiOverview struct {
	Name      string   `json:"name"`
	Languages []string `json:"languages"`
}

func setBmmEnvironment(client *resty.Client, environment apiv1.BmmEnvironment) {
	if environment == apiv1.BmmEnvironment_Integration {
		client.BaseURL = os.Getenv("BMM_INT_BASE_URL")
	} else {
		client.BaseURL = os.Getenv("BMM_BASE_URL")
	}
}

func (a BMMApi) GetAlbums(_ context.Context, req *connect.Request[apiv1.GetAlbumsRequest]) (*connect.Response[apiv1.AlbumsList], error) {
	if !PermissionsForEmail(getEmail(req)).CanUpload() {
		return nil, connect.NewError(403, fmt.Errorf("not authorized"))
	}

	setBmmEnvironment(a.client, req.Msg.Environment)

	albumsReq := a.client.R().
		SetAuthToken(a.token.GetAccessToken()).
		SetResult(&[]BMMItem{})

	albumsResponse, err := albumsReq.Get(fmt.Sprintf("/album/published/%d/", req.Msg.Year))
	if err != nil {
		return nil, err
	}

	albums := albumsResponse.Result().(*[]BMMItem)
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

func (a BMMApi) GetAlbumTracks(_ context.Context, req *connect.Request[apiv1.GetAlbumTracksRequest]) (*connect.Response[apiv1.TracksList], error) {
	permissions := PermissionsForEmail(getEmail(req))
	if !permissions.CanUpload() {
		return nil, connect.NewError(403, fmt.Errorf("not authorized"))
	}

	setBmmEnvironment(a.client, req.Msg.Environment)

	tracksReq := a.client.R().
		SetAuthToken(a.token.GetAccessToken()).
		SetResult(&BMMItem{})

	res, err := tracksReq.Get(fmt.Sprintf("/album/%s", req.Msg.AlbumId))
	if err != nil {
		return nil, err
	}

	album := res.Result().(*BMMItem)

	tracks := &apiv1.TracksList{}
	for _, track := range album.Tracks {
		langs := lo.Intersect(track.Languages, permissions.Bmm.Languages)
		tracks.Tracks = append(tracks.Tracks, &apiv1.BMMTrack{
			Id:          strconv.Itoa(track.ID),
			Title:       track.Title,
			PublishedAt: timestamppb.New(track.PublishedAt),
			Languages:   languageListToApi(langs),
		})
	}

	return connect.NewResponse(tracks), nil
}

func (a BMMApi) GetPodcastTracks(_ context.Context, req *connect.Request[apiv1.GetPodcastTracksRequest]) (*connect.Response[apiv1.TracksList], error) {
	permissions := PermissionsForEmail(getEmail(req))
	if !permissions.CanUpload() {
		return nil, connect.NewError(403, fmt.Errorf("not authorized"))
	}

	setBmmEnvironment(a.client, req.Msg.Environment)

	tracksReq := a.client.R().
		SetAuthToken(a.token.GetAccessToken()).SetResult(&[]BMMItem{})

	res, err := tracksReq.Get(fmt.Sprintf("/track?tags=%s&size=%d&unpublished=show", url.QueryEscape(req.Msg.PodcastTag), req.Msg.Limit))
	if err != nil {
		return nil, err
	}

	tracks := *(res.Result().(*[]BMMItem))
	tracksOut := &apiv1.TracksList{}
	for _, track := range tracks {
		langs := lo.Intersect(track.Languages, permissions.Bmm.Languages)
		tracksOut.Tracks = append(tracksOut.Tracks, &apiv1.BMMTrack{
			Id:          strconv.Itoa(track.ID),
			Title:       track.Title,
			PublishedAt: timestamppb.New(track.PublishedAt),
			Languages:   languageListToApi(langs),
		})
	}

	return connect.NewResponse(tracksOut), nil
}

func languageListToApi(languages []string) *apiv1.LanguageList {
	languagesOut := &apiv1.LanguageList{}

	sort.Strings(languages)

	for _, l := range languages {
		languagesOut.Languages = append(languagesOut.Languages, &apiv1.Language{Code: l, FlagEmoji: EmojiForLanguage(l)})
	}

	return languagesOut
}

func (a BMMApi) GetLanguages(_ context.Context, req *connect.Request[apiv1.GetAvailableLanguagesRequest]) (*connect.Response[apiv1.LanguageList], error) {
	setBmmEnvironment(a.client, req.Msg.Environment)

	overviewRequest := a.client.R().SetAuthToken(a.token.GetAccessToken()).SetResult(&BMMApiOverview{})
	res, err := overviewRequest.Get("/")
	if err != nil {
		return nil, err
	}

	overview := res.Result().(*BMMApiOverview)
	return connect.NewResponse(languageListToApi(overview.Languages)), nil
}
