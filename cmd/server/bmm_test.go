package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func Test_getToken(t *testing.T) {
	if os.Getenv("BMM_AUTH0_BASE_URL") == "" {
		t.Skip("Required ENV variables not set for getting token")
	}

	tokenBaseURL := os.Getenv("BMM_AUTH0_BASE_URL")
	clientID := os.Getenv("BMM_CLIENT_ID")
	clientSecret := os.Getenv("BMM_CLIENT_SECRET")
	audience := os.Getenv("BMM_AUDIENCE")

	res, err := getToken(tokenBaseURL, clientID, clientSecret, audience)
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
}

func Test_UnmarshallAlbumWithTracks(t *testing.T) {
	data, err := os.ReadFile("testdata/album_with_tracks.json")
	assert.NoError(t, err)

	album := &BMMItem{}
	err = json.Unmarshal(data, album)
	assert.NoError(t, err)
	assert.NotEmpty(t, album)

	assert.Equal(t, "album", album.Type)
	assert.Equal(t, 6, len(album.Tracks))
}
