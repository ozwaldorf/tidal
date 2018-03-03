package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/The5heepDev/tidal"
	tui "github.com/marcusolsson/tui-go"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

var t *tidal.Tidal
var albumResults []tidal.Album
var trackResults []tidal.Track
var ui tui.UI
var status *tui.StatusBar
var progress *tui.Progress
var infoBox *tui.Box
var dl *tui.List
var downQueue = make(chan tidal.Track, 512)
var downList []tidal.Track
var todo, done int
var current = ""
var tq = make(chan string, 16)

const logo = `   __  _     __      __                        
  / /_(_)___/ /____ / /      ______ __   _____ 
 / __/ / __  / __  / / | /| / / __  / | / / _ \
/ /_/ / /_/ / /_/ / /| |/ |/ / /_/ /| |/ /  __/
\__/_/\__,_/\__,_/_/ |__/|__/\__,_/ |___/\___/ `

func main() {

	user := tui.NewEntry()
	user.SetFocused(true)

	password := tui.NewEntry()

	form := tui.NewGrid(0, 0)
	form.AppendRow(tui.NewLabel("Username"), tui.NewLabel("Password"))
	form.AppendRow(user, password)

	login := tui.NewButton("[Login]")

	buttons := tui.NewHBox(
		tui.NewSpacer(),
		tui.NewPadder(1, 0, login),
	)
	msg := tui.NewStatusBar("   yeet yeet")
	window := tui.NewVBox(
		tui.NewPadder(10, 1, tui.NewLabel(logo)),
		tui.NewPadder(12, 0, msg),
		tui.NewPadder(1, 1, form),
		buttons,
	)
	window.SetBorder(true)

	wrapper := tui.NewVBox(
		tui.NewSpacer(),
		window,
		tui.NewSpacer(),
	)
	content := tui.NewHBox(tui.NewSpacer(), wrapper, tui.NewSpacer())
	loginPage := tui.NewVBox(content, tui.NewStatusBar("[Ctrl+Q: Quit]  [Tab: Cycle input]"))
	tui.DefaultFocusChain.Set(user, password, login)

	var err error
	ui, err = tui.New(loginPage)
	if err != nil {
		log.Fatal(err)
	}

	win := tui.NewTable(0, 0)
	win.SetColumnStretch(0, 2)
	win.SetColumnStretch(1, 4)
	win.SetColumnStretch(2, 4)
	win.SetColumnStretch(3, 1)
	win.SetColumnStretch(4, 1)
	win.SetColumnStretch(5, 1)
	win.SetSizePolicy(tui.Maximum, tui.Maximum)
	win.AppendRow(
		tui.NewLabel("ARTIST"),
		tui.NewLabel("ALBUM"),
		tui.NewLabel("TITLE"),
		tui.NewLabel("NUMBER"),
		tui.NewLabel("EXPLICIT"),
		tui.NewLabel(""),
	)
	win.SetSelected(1)
	libBox := tui.NewVBox(
		win,
		tui.NewSpacer(),
	)
	libBox.SetBorder(true)
	libBox.SetTitle("=[ Search:  ]=")

	dl = tui.NewList()
	dlS := tui.NewScrollArea(dl)
	dlBox := tui.NewVBox(dlS)
	dlBox.SetBorder(true)
	dlBox.SetTitle("=[ Downloads ]=")

	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)
	inputBox := tui.NewVBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)
	inputBox.SetTitle("=[Search Tracks]=")
	searchType := 0

	progress = tui.NewProgress(100)
	progress.SetSizePolicy(tui.Expanding, tui.Maximum)
	infoBox = tui.NewHBox(progress)
	infoBox.SetBorder(true)
	infoBox.SetSizePolicy(tui.Expanding, tui.Maximum)
	infoBox.SetTitle("=[ 0 | 0 ][]=")

	help := tui.NewLabel("[Tab: Switch Search Type]   [CtrlD: Switch View]   [CtrlQ: Quit]")
	help.SetSizePolicy(tui.Expanding, tui.Maximum)

	v := []*tui.Box{
		tui.NewVBox(
			infoBox,
			libBox,
			inputBox,
			help,
		),
		tui.NewVBox(
			infoBox,
			dlBox,
			help,
		),
	}

	cur := 0

	login.OnActivated(func(b *tui.Button) {
		var err error
		t, err = tidal.New(user.Text(), password.Text())
		if err != nil {
			log.Fatal(err)
		}
		if t.SessionID != "" {
			ui.SetWidget(v[0])
			ui.SetKeybinding("Up", func() {
				if cur == 1 {
					dlS.Scroll(0, -1)
				}
			})
			ui.SetKeybinding("Down", func() {
				if cur == 1 {
					dlS.Scroll(0, 1)
				}
			})
			ui.SetKeybinding("PgUp", func() {
				if cur == 0 {
					win.Select(win.Selected() - 10)
				} else {
					dlS.Scroll(0, -10)
				}
			})
			ui.SetKeybinding("PgDn", func() {
				if cur == 0 {
					win.Select(win.Selected() + 10)
				} else {
					dlS.Scroll(0, 10)
				}
			})
			ui.SetKeybinding("Ctrl+D", func() {
				cur = int(math.Abs(float64(cur - 1))) // toggle between
				ui.SetWidget(v[cur])
			})
			ui.SetKeybinding("Tab", func() {
				searchType = int(math.Abs(float64(searchType - 1))) // toggle between
				if searchType == 0 {
					inputBox.SetTitle("=[Search Tracks]=")
				} else {
					inputBox.SetTitle("=[Search Albums]=")
				}
			})
			ui.SetKeybinding("Enter", func() {
				if cur != 1 {
					if input.Text() != "" {
						libBox.SetTitle("=[ Search: " + input.Text() + " ]=")
						win.RemoveRows()
						win.AppendRow(
							tui.NewLabel("ARTIST"),
							tui.NewLabel("ALBUM"),
							tui.NewLabel("TITLE"),
							tui.NewLabel("NUMBER"),
							tui.NewLabel("EXPLICIT"),
							tui.NewLabel(""),
						)
						win.SetSelected(1)
						var err error
						switch searchType {
						case 0:
							trackResults, err = t.SearchTracks(input.Text(), fmt.Sprintf("%d", libBox.Size().Y))
							if err != nil {
								log.Fatal(err)
							}
							for _, v := range trackResults {
								win.AppendRow(
									tui.NewLabel(v.Artists[0].Name),
									tui.NewLabel(v.Album.Title),
									tui.NewLabel(v.Title),
									tui.NewLabel(v.TrackNumber.String()),
									tui.NewLabel(fmt.Sprintf("%t", v.Explicit)),
								)
							}

						case 1:
							albumResults, err = t.SearchAlbums(input.Text(), fmt.Sprintf("%d", libBox.Size().Y))
							if err != nil {
								log.Fatal(err)
							}
							for _, v := range albumResults {
								win.AppendRow(
									tui.NewLabel(v.Artists[0].Name),
									tui.NewLabel(v.Title),
									tui.NewLabel(""),
									tui.NewLabel(v.NumberOfTracks.String()),
									tui.NewLabel(fmt.Sprintf("%t", v.Explicit)),
								)
							}
						}
						input.SetText("")
					} else if len(albumResults) > 0 || len(trackResults) > 0 {
						go func() {
							switch searchType {
							case 0:
								todo++
								v := trackResults[win.Selected()-1]
								dl.AddItems(fmt.Sprintf("%s - %s", v.Artists[0].Name, v.Title))
								downQueue <- v
							case 1:
								d, err := t.GetAlbumTracks(albumResults[win.Selected()-1].ID.String())
								if err != nil {
									log.Fatal(err)
								}
								todo += len(d)
								for _, v := range d {
									dl.AddItems(fmt.Sprintf("%s - %s", v.Artists[0].Name, v.Title))
									tq <- fmt.Sprintf("=[ (%d/%d) %s ]=", done, todo, current)
									downQueue <- v
								}
							}
						}()
					}
				}
			})
		} else {
			msg.SetText("wrong username or password.")
		}
	})

	ui.SetKeybinding("Ctrl+Q", func() { ui.Quit() })

	win.OnSelectionChanged(func(t *tui.Table) {
		if t.Selected() >= win.Size().Y {
			t.SetSelected(win.Size().Y - 1)
		} else if t.Selected() < 1 {
			t.SetSelected(1)
		}
	})

	go func() {
		for v := range downQueue {
			done++
			downloadTrack(v, "LOSSLESS")
		}
	}()
	go func() {
		for v := range tq {
			ui.Update(func() {
				infoBox.SetTitle(v)
			})
		}
	}()

	if err := ui.Run(); err != nil {
		panic(err)
	}
}

// DownloadTrack (id of track, quality of file)
func downloadTrack(tr tidal.Track, q string) {
	dirs := clean(tr.Artists[0].Name) + "/" + clean(tr.Album.Title)
	path := dirs + "/" + clean(tr.Artists[0].Name) + " - " + clean(tr.Title)
	current = path
	tq <- fmt.Sprintf("=[ (%d/%d) %s ]=", done, todo, current)
	os.MkdirAll(dirs, os.ModePerm)
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}

	u, err := t.GetStreamURL(tr.ID.String(), q)
	if err != nil {
		log.Fatal(err)
	}
	res, err := http.Get(u)
	if err != nil {
		log.Fatal(err)
	}
	r := newProxy(res.Body, int(res.ContentLength))
	io.Copy(f, r)
	res.Body.Close()
	f.Close()
	r.Close()

	err = enc(path, tr.Title, tr.Artists[0].Name, tr.Album.Title, tr.TrackNumber.String())
	if err != nil {
		panic(err)
	}
	os.Remove(path)
}

func clean(s string) string {
	return strings.Replace(s, "/", "\u2215", -1)
}

func enc(src, title, artist, album, num string) error {
	// Decode FLAC file.
	stream, err := flac.ParseFile(src)
	if err != nil {
		return err
	}

	// Add custom vorbis comment.
	for _, block := range stream.Blocks {
		if comment, ok := block.Body.(*meta.VorbisComment); ok {
			comment.Tags = append(comment.Tags, [2]string{"TITLE", title})
			comment.Tags = append(comment.Tags, [2]string{"ARTIST", artist})
			comment.Tags = append(comment.Tags, [2]string{"ALBUMARTIST", artist})
			comment.Tags = append(comment.Tags, [2]string{"ALBUM", album})
			comment.Tags = append(comment.Tags, [2]string{"TRACKNUMBER", num})
		}
	}

	// Encode FLAC file.
	f, err := os.Create(src + ".flac")
	if err != nil {
		return err
	}
	err = flac.Encode(f, stream)
	f.Close()
	stream.Close()
	return err
}

/* little bit to proxy the reader and update the progress bar */
type proxyRead struct {
	io.Reader
	t, l int
}

func newProxy(r io.Reader, l int) *proxyRead {
	return &proxyRead{r, 0, l}
}

func (r *proxyRead) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.t += n
	ui.Update(func() {
		progress.SetCurrent(r.t)
		progress.SetMax(r.l)
	})
	return
}

// Close the reader when it implements io.Closer
func (r *proxyRead) Close() (err error) {
	ui.Update(func() {
		progress.SetCurrent(0)
		progress.SetMax(1)
	})
	if closer, ok := r.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return
}
