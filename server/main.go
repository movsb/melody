package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

type BodyLike struct {
	Location      string `json:"location"`
	Title         string `json:"title"`
	ChannelName   string `json:"channelName"`
	BadgeIsArtist bool   `json:"badgeIsArtist"`
	Liked         bool   `json:"liked"`
	Description   string `json:"description"`
	WatchMetadata string `json:"watchMetadata"`
	HasMusicInfo  bool   `json:"hasMusicInfo"`
}

func main() {
	if err := os.Chdir("/data"); err != nil {
		panic(err)
	}
	mgr := NewManager(`list.yaml`)
	http.HandleFunc(`/v1/youtube:like`, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Add(`Access-Control-Allow-Origin`, `*`)
			w.Header().Add(`Access-Control-Allow-Credentials`, `true`)
			w.Header().Add(`Access-Control-Allow-Methods`, `GET, POST, OPTIONS`)
			w.Header().Add(`Access-Control-Allow-Headers`, `Content-Type`)
			return
		}
		w.Header().Add(`Access-Control-Allow-Origin`, `*`)

		var like BodyLike
		if err := json.NewDecoder(r.Body).Decode(&like); err != nil {
			panic(err)
		}

		mgr.setLike(like.Location, mgr.shouldLike(&like))
	})
	http.HandleFunc(`/v1/youtube:downloaded`, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Add(`Access-Control-Allow-Origin`, `*`)
			w.Header().Add(`Access-Control-Allow-Credentials`, `true`)
			w.Header().Add(`Access-Control-Allow-Methods`, `GET, POST, OPTIONS`)
			w.Header().Add(`Access-Control-Allow-Headers`, `Content-Type`)
			return
		}
		w.Header().Add(`Access-Control-Allow-Origin`, `*`)

		// only location
		var like BodyLike
		if err := json.NewDecoder(r.Body).Decode(&like); err != nil {
			panic(err)
		}

		json.NewEncoder(w).Encode(&BodyGetDownloaded{Done: mgr.downloaded(like.Location)})

	})
	http.ListenAndServe(`:80`, nil)
}

type BodyGetDownloaded struct {
	Done bool `json:"done"`
}

type Manager struct {
	items     map[string]*Item
	lockItems sync.Mutex
	listFile  string
}

type Item struct {
	Done bool `yaml:"done"`
}

func (m *Manager) shouldLike(like *BodyLike) bool {
	if !like.Liked {
		return false
	}

	if like.BadgeIsArtist {
		return true
	}
	if like.HasMusicInfo {
		return true
	}

	for _, s := range []string{like.Title, like.ChannelName, like.Description, like.WatchMetadata} {
		ls := strings.ToLower(s)
		switch {
		case strings.Contains(ls, "music"):
			return true
		case strings.Contains(ls, "piano"):
			return true
		case strings.Contains(ls, "soundtrack"):
			return true
		}
	}

	return false
}

func NewManager(listFile string) *Manager {
	items := make(map[string]*Item)
	stat, err := os.Stat(listFile)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	if err == nil && stat.Size() > 0 {
		fp, err := os.Open(listFile)
		if err != nil {
			panic(err)
		}
		defer fp.Close()
		if err := yaml.NewDecoder(fp).Decode(&items); err != nil {
			panic(err)
		}
	}

	return &Manager{
		listFile: listFile,
		items:    items,
	}
}

func (m *Manager) downloaded(link string) bool {
	id := m.getID(link)

	m.lockItems.Lock()
	defer m.lockItems.Unlock()

	if item, ok := m.items[id]; ok && item.Done {
		return true
	}

	return false
}

func (m *Manager) saveListFile() {
	fp, err := os.Create(m.listFile)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	if err := yaml.NewEncoder(fp).Encode(m.items); err != nil {
		panic(err)
	}
}

func (m *Manager) setLike(link string, like bool) {
	if link == "" {
		panic("empty link")
	}

	m.lockItems.Lock()
	defer m.lockItems.Unlock()

	// 只会下载不存在的，如果以前没下完，不会下。
	if _, ok := m.items[m.getID(link)]; !ok && like {
		go m.download(link)
		return
	}

	if !like {
		go m.remove(link)
	}
}

func (m *Manager) download(link string) {
	id := m.getID(link)

	m.lockItems.Lock()
	if _, ok := m.items[id]; !ok {
		m.items[id] = &Item{}
	}
	m.lockItems.Unlock()

	m.saveListFile()

	log.Println("进入下载", link)

	cmd := exec.Command(`yt-dlp`, `--add-metadata`, `--embed-thumbnail`, `--embed-subs`, `--no-playlist`, `--force-ipv4`, `--proxy`, `socks5://192.168.1.86:1080`, link)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ok := false
	if err := cmd.Run(); err != nil {
		log.Println("下载失败", link, err)
		ok = false
	} else {
		ok = true
	}

	m.lockItems.Lock()
	defer m.lockItems.Unlock()
	m.items[id].Done = ok
	m.saveListFile()
}

func (m *Manager) getID(link string) string {
	var id string
	u, err := url.Parse(link)
	if err != nil {
		panic(err)
	}

	if strings.ToLower(u.Host) == "www.youtube.com" {
		id = u.Query().Get("v")
	} else if strings.ToLower(u.Host) == "youtu.be" {
		id = u.Path
	}

	if id == "" {
		panic("no id was found for " + link)
	}

	return id
}

// https://www.youtube.com/watch?v=fU2NJrXkMPA
// https://youtu.be/gOcQP_Gi9r8
func (m *Manager) remove(link string) {
	id := m.getID(link)

	paths, err := filepath.Glob(fmt.Sprintf(`*[%s]*`, id))
	if err != nil {
		panic(err)
	}

	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			panic(err)
		}
		fmt.Println("删除文件", path)
	}

	m.lockItems.Lock()
	defer m.lockItems.Unlock()

	delete(m.items, id)

	m.saveListFile()
}
