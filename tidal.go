package tidal

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

const baseurl = "https://api.tidalhifi.com/v1/"
const clientVersion = "1.9.1" // ayy that's the golang version too!
const token = "kgsOOmYk3zShYrNP"

var cookieJar, _ = cookiejar.New(nil)
var c = &http.Client{
	Jar: cookieJar, // I stole the cookie from the cookie jar
}

func (tidal *Tidal) get(dest string, query *url.Values, s interface{}) {
	req, err := http.NewRequest("GET", baseurl+dest, nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("X-Tidal-SessionID", tidal.SessionID)
	query.Add("countryCode", tidal.CountryCode)
	req.URL.RawQuery = query.Encode()
	res, _ := c.Do(req)
	defer res.Body.Close()
	d := json.NewDecoder(res.Body)
	d.Decode(&s)
}

// GetStreamURL func
func (tidal *Tidal) GetStreamURL(id, q string) string {
	var s map[string]interface{}
	tidal.get("tracks/"+id+"/streamUrl", &url.Values{
		"soundQuality": {q},
	}, &s)
	return s["url"].(string)
}

// GetAlbumTracks func
func (tidal *Tidal) GetAlbumTracks(id string) []Track {
	var s struct {
		Items []Track `json:"items"`
	}
	tidal.get("albums/"+id+"/tracks", &url.Values{}, &s)
	return s.Items
}

// GetPlaylistTracks func
func (tidal *Tidal) GetPlaylistTracks(id string) []Track {
	var s struct {
		Items []Track `json:"items"`
	}
	tidal.get("playlists/"+id+"/tracks", &url.Values{}, &s)
	return s.Items
}

// SearchTracks func
func (tidal *Tidal) SearchTracks(d, l string) []Track {
	var s Search
	tidal.get("search", &url.Values{
		"query": {d},
		"types": {"TRACKS"},
		"limit": {l},
	}, &s)
	return s.Tracks.Items
}

// SearchAlbums func
func (tidal *Tidal) SearchAlbums(d, l string) []Album {
	var s Search
	tidal.get("search", &url.Values{
		"query": {d},
		"types": {"ALBUMS"},
		"limit": {l},
	}, &s)
	return s.Albums.Items
}

// SearchArtists func
func (tidal *Tidal) SearchArtists(d, l string) []Artist {
	var s Search
	tidal.get("search", &url.Values{
		"query": {d},
		"types": {"ARTISTS"},
		"limit": {l},
	}, &s)
	return s.Artists.Items
}

// helper function to generate a uuid
func uuid() string {
	b := make([]byte, 16)
	rand.Read(b[:])
	b[8] = (b[8] | 0x40) & 0x7F
	b[6] = (b[6] & 0xF) | (4 << 4)
	return fmt.Sprintf("%x", b)
}

// New func
func New(user, pass string) *Tidal {
	query := url.Values{
		"username":        {user},
		"password":        {pass},
		"token":           {token},
		"clientUniqueKey": {uuid()},
		"clientVersion":   {clientVersion},
	}
	res, _ := http.PostForm(baseurl+"login/username", query)
	defer res.Body.Close()
	d := json.NewDecoder(res.Body)
	var t Tidal
	d.Decode(&t)
	return &t
}
