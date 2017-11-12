package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
	"github.com/the5heepdev/tidal"
)

func main() {
	t := tidal.New("", "") // put your username and pass
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		d := t.SearchTracks(string(text))
		downloadTrack(t, d[0], "LOSSLESS")
	}
}

func downloadAlbum(t *tidal.Tidal, a tidal.Album) {
	d := t.GetAlbumTracks(a.ID.String())
	for _, v := range d {
		downloadTrack(t, v, "LOSSLESS")
	}
}

// DownloadTrack (id of track, quality of file)
func downloadTrack(tid *tidal.Tidal, t tidal.Track, q string) {
	u := tid.GetStreamURL(t.ID.String(), q)
	res, err := http.Get(u)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	dirs := clean(t.Artists[0].Name) + "/" + clean(t.Album.Title)
	path := dirs + "/" + clean(t.Artists[0].Name) + " - " + clean(t.Title)
	os.MkdirAll(dirs, os.ModePerm)
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	written := 0
	for i := 0; written < int(res.ContentLength); i++ {
		fmt.Printf("\r[%3.0f] downloading %s", (float64(written)/float64(res.ContentLength))*100, path)
		buf := make([]byte, 2048)
		io.ReadFull(res.Body, buf)
		n, err := f.Write(buf)
		if err != nil {
			fmt.Println(err)
			return
		}
		written += n
	}
	f.Close()

	err = enc(path, t.Title, t.Artists[0].Name, t.Album.Title, t.TrackNumber)
	if err != nil {
		fmt.Println(err)
	}

}

func clean(s string) string {
	return strings.Replace(s, "/", "\u2215", -1)
}

func enc(src, title, artist, album string, num int) error {
	// Decode FLAC file.
	stream, err := flac.ParseFile(src)
	if err != nil {
		return err
	}
	defer stream.Close()

	// Add custom vorbis comment.
	for _, block := range stream.Blocks {
		if comment, ok := block.Body.(*meta.VorbisComment); ok {
			comment.Tags = append(comment.Tags, [2]string{"TITLE", title})
			comment.Tags = append(comment.Tags, [2]string{"ARTIST", artist})
			comment.Tags = append(comment.Tags, [2]string{"ALBUMARTIST", artist})
			comment.Tags = append(comment.Tags, [2]string{"ALBUM", album})
			comment.Tags = append(comment.Tags, [2]string{"TRACKNUMBER", string(num)})
		}
	}

	// Encode FLAC file.
	f, err := os.Create(src + ".flac")
	if err != nil {
		return err
	}
	defer f.Close()
	err = flac.Encode(f, stream)
	if err != nil {
		return err
	}

	return os.Remove(src)
}
