package main

import (
	"bytes"
	"embed"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

//go:embed melody.user.js
var fs embed.FS

func main() {
	if err := os.Chdir("/data"); err != nil {
		panic(err)
	}

	mgr := NewManager(`list.yaml`)

	fsHandler := func(hfs http.FileSystem) http.Handler {
		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(hfs))
		now := time.Now().In(time.FixedZone(`China`, 8*60*60)).Format(`2006.1.2.15.4.5`)
		mux.HandleFunc("/melody.user.js", func(w http.ResponseWriter, r *http.Request) {
			f, err := hfs.Open(`melody.user.js`)
			if err != nil {
				panic(err)
			}
			w.Header().Set(`Content-Type`, `text/javascript`)
			allBytes, _ := io.ReadAll(f)
			all := bytes.Replace(allBytes, []byte(`VERSION_PLACEHOLDER`), []byte(now), 1)
			w.Write(all)
		})
		return mux
	}

	apiHandler := func(mgr *Manager) http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc(`/v1/youtube:downloaded`, func(w http.ResponseWriter, r *http.Request) {
			url := r.URL.Query().Get(`url`)
			fmt.Fprint(w, mgr.getStatus(url))
		})
		mux.HandleFunc(`/v1/youtube:download`, func(w http.ResponseWriter, r *http.Request) {
			url := r.URL.Query().Get(`url`)
			// 先清除可能残留的临时文件
			mgr.remove(url)
			// 等待下载结束
			wait := r.URL.Query().Get(`wait`)
			if wait == `1` || wait == `true` {
				mgr.download(url)
			} else {
				go mgr.download(url)
			}
		})
		mux.HandleFunc(`/v1/youtube:delete`, func(w http.ResponseWriter, r *http.Request) {
			url := r.URL.Query().Get(`url`)
			mgr.remove(url)
		})
		return mux
	}

	corsHandler := func(api http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Add(`Access-Control-Allow-Origin`, `*`)
				w.Header().Add(`Access-Control-Allow-Credentials`, `true`)
				w.Header().Add(`Access-Control-Allow-Methods`, `GET, POST, OPTIONS`)
				w.Header().Add(`Access-Control-Allow-Headers`, `Content-Type`)
				return
			}
			w.Header().Add(`Access-Control-Allow-Origin`, `*`)
			api.ServeHTTP(w, r)
		}
	}

	http.Handle("/static/", http.StripPrefix("/static", fsHandler(http.FS(fs))))
	http.Handle("/v1/", corsHandler(apiHandler(mgr)))

	http.ListenAndServe(`:80`, nil)
}

type BodyGetDownloaded struct {
	Status bool `json:"status"`
}

type Manager struct {
	items     map[string]*Item
	lockItems sync.Mutex
	listFile  string
}

type Item struct {
	isDownloading bool `json:"-"`
	Done          bool `yaml:"done"`
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

func (m *Manager) getStatus(link string) string {
	id := m.getID(link)

	m.lockItems.Lock()
	defer m.lockItems.Unlock()

	if item, ok := m.items[id]; ok {
		if item.isDownloading {
			return "Downloading"
		}
		if item.Done {
			return "Downloaded"
		}
		return "Failed"
	}
	return "Not Downloaded"
}

func (m *Manager) saveListFile() {
	m.lockItems.Lock()
	defer m.lockItems.Unlock()

	name := m.listFile + `.tmp`
	defer os.Remove(name)
	fp, err := os.Create(name)
	if err != nil {
		panic(err)
	}

	enc := yaml.NewEncoder(fp)
	if err := enc.Encode(m.items); err != nil {
		panic(err)
	}
	if err := enc.Close(); err != nil {
		panic(err)
	}
	if err := fp.Close(); err != nil {
		panic(err)
	}

	if err := os.Rename(name, m.listFile); err != nil {
		panic(err)
	}
}

func (m *Manager) download(link string) {
	id := m.getID(link)

	m.saveListFile()

	log.Println("进入下载", link)

	m.lockItems.Lock()
	item, ok := m.items[id]
	if !ok {
		item = &Item{}
		m.items[id] = item
	}
	if item.isDownloading {
		m.lockItems.Unlock()
		return
	}
	item.isDownloading = true
	m.lockItems.Unlock()

	cmd := exec.Command(`yt-dlp`, `--add-metadata`, `--embed-thumbnail`, `--embed-subs`, `--no-playlist`, `--force-ipv4`, `--no-check-certificates`, `--proxy`, `socks5://192.168.1.86:1080`, link)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Println("下载失败", link, err)
		ok = false
	} else {
		ok = true
	}

	defer m.saveListFile()
	m.lockItems.Lock()
	defer m.lockItems.Unlock()
	item.isDownloading = false
	// 如果目录下存在 ID 相关的临时文件，则认为没有成功下载。
	paths, err := filepath.Glob(fmt.Sprintf(`*\[%s\].*.part`, id))
	if err != nil {
		panic(err)
	}
	item.Done = ok && len(paths) == 0
}

var rePlainId = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)

// https://www.youtube.com/watch?v=fU2NJrXkMPA
// https://youtu.be/gOcQP_Gi9r8
// JGwWNGJdvx8
func (m *Manager) getID(link string) string {
	if rePlainId.MatchString(link) {
		return link
	}

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

func (m *Manager) remove(link string) {
	id := m.getID(link)

	paths, err := filepath.Glob(fmt.Sprintf(`*\[%s\].*`, id))
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
	delete(m.items, id)
	m.lockItems.Unlock()

	m.saveListFile()
}
