package tidal

import "encoding/json"

// Tidal api struct
type Tidal struct {
	SessionID   string      `json:"sessionID"`
	CountryCode string      `json:"countryCode"`
	UserID      json.Number `json:"userId"`
}

// Artist struct
type Artist struct {
	ID         json.Number `json:"id"`
	Name       string      `json:"name"`
	Popularity int         `json:"popularity"`
}

// Album struct
type Album struct {
	Artists        []Artist    `json:"artists,omitempty"`
	Title          string      `json:"title"`
	ID             json.Number `json:"id"`
	NumberOfTracks json.Number `json:"numberOfTracks"`
	Explicit       bool        `json:"explicit,omitempty"`
	Copyright      string      `json:"copyright,omitempty"`
}

// Track struct
type Track struct {
	Artists     []Artist    `json:"artists"`
	Album       Album       `json:"album"`
	Title       string      `json:"title"`
	ID          json.Number `json:"id"`
	Explicit    bool        `json:"explicit"`
	Copyright   string      `json:"copyright"`
	Popularity  int         `json:"popularity"`
	TrackNumber json.Number `json:"trackNumber"`
}

// Search struct
type Search struct {
	Albums struct {
		Items []Album `json:"items"`
	} `json:"albums"`
	Artists struct {
		Items []Artist `json:"items"`
	} `json:"artists"`
	Tracks struct {
		Items []Track `json:"items"`
	} `json:"tracks"`
}
