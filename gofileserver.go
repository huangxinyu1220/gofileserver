package main

import (
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type GoFileServer struct {
	Root            string
	Upload          bool
	Delete          bool
	indexMap		map[string]FileInfo
	m				*mux.Router
}

type FileInfo struct {
	Name			string			`json:"name"`
	Path			string			`json:"path"`
	Size 			int64			`json:"size"`
	ModTime			int64			`json:"mtime"`
	IsDir			bool			`json:"is_dir"`
}

func NewGoFileServer(root string) *GoFileServer {
	if root == "" {
		root = "./"
	}
	root = filepath.ToSlash(root)
	if !strings.HasSuffix(root, "/") {
		root = root + "/"
	}
	m := mux.NewRouter()
	s := &GoFileServer{
		Root:   root,
		m:      m,
	}
	go func() {
		for	{
			s.initIndexMap()
			time.Sleep(5 * time.Minute)
		}
	}()
	m.HandleFunc("/-/status", s.status).Methods("GET")
	m.HandleFunc("/-/mkdir/{path:.*}", s.mkdir).Methods("POST")
	m.HandleFunc("/{path:.*}", s.index).Methods("GET")
	m.HandleFunc("/{path:.*}", s.delete).Methods("DELETE")
	m.HandleFunc("/{path:.*}", s.upload).Methods("POST")
	return s
}

func (s *GoFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func (s *GoFileServer) initIndexMap() {
	indexMap := make(map[string]FileInfo)
	filepath.Walk(s.Root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			relPath, err := filepath.Rel(s.Root, path)
			if err != nil {
				return err
			}
			relPath = filepath.ToSlash(relPath)
			fileInfo := FileInfo{
				Name:    relPath,
				Path:    relPath,
				Size:    info.Size(),
				ModTime: info.ModTime().UnixNano() / 1e6,
				IsDir:   info.IsDir(),
			}
			indexMap[relPath] = fileInfo
		}
		return nil
	})
	s.indexMap = indexMap
}

func (s *GoFileServer) searchIndex (rootPath string, text string) []FileInfo{
	fileList := make([]FileInfo, 0)
	length := len(rootPath)
	for path, fileInfo := range s.indexMap {
		if strings.HasPrefix(path, rootPath) && strings.Contains(filepath.Base(path), text){
			if length > 0 {
				fileInfo.Name = fileInfo.Name[length+1:]
			}
			fileList = append(fileList, fileInfo)
		}
	}
	return fileList
}

func (s *GoFileServer) status(w http.ResponseWriter, r *http.Request) {
	data, _ := json.MarshalIndent(s, "", "    ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *GoFileServer) mkdir(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	name := r.FormValue("name")
	matched, err := regexp.MatchString("^[0-9a-zA-Z_\\.]+$", name)
	if err != nil {
		http.Error(w, err.Error(), 400)
		log.Println(err)
		return
	}
	if !matched {
		http.Error(w, "Dirname contains illegal characters!", 400)
		log.Println(err)
		return
	}
	err = os.Mkdir(filepath.Join(s.Root, path, name), 0755)
	if err != nil {
		http.Error(w, err.Error(), 400)
		log.Println(err)
	}
}

func (s *GoFileServer) index(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("json") == "true" {
		s.jsonList(w, r)
		return
	}
	path := mux.Vars(r)["path"]
	realPath := filepath.Join(s.Root, path)
	if isDir(realPath) {
		if r.FormValue("download") == "true" {
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", "attachment; filename=" + strconv.Quote(filepath.Base(path)+".zip"))
			compressZipFile(w, realPath)
		} else {
			renderHTML(w, "index.html", s)
		}
	} else {
		if r.FormValue("download") == "true" {
			w.Header().Set("Content-Disposition", "attachment; filename=" + strconv.Quote(filepath.Base(path)))
		}
		http.ServeFile(w, r, realPath)
	}
}

func (s *GoFileServer) jsonList(w http.ResponseWriter, r *http.Request) {
	fileList := make([]FileInfo, 0)
	path := mux.Vars(r)["path"]
	if r.FormValue("search") != "" {
		searchText := r.FormValue("search")
		fileList = s.searchIndex(path, searchText)
	} else {
		localPath := filepath.Join(s.Root, path)
		infos, err := ioutil.ReadDir(localPath)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Println(err.Error())
			return
		}
		filePathMap := make(map[string]os.FileInfo)
		for _, info := range infos {
			filePathMap[filepath.Join(path, info.Name())] = info
		}
		for path, info := range filePathMap {
			fileInfo := FileInfo{
				Name:    info.Name(),
				Path:    path,
				Size:    info.Size(),
				ModTime: info.ModTime().UnixNano() / 1e6,
				IsDir:   info.IsDir(),
			}
			fileList = append(fileList, fileInfo)
		}
	}
	data, err := json.Marshal(fileList)
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *GoFileServer) delete(w http.ResponseWriter, r *http.Request) {
	log.Println("get delete request")
	path := mux.Vars(r)["path"]
	err := os.RemoveAll(filepath.Join(s.Root, path))
	if err != nil {
		http.Error(w, err.Error(), 400)
		log.Println(err.Error())
	}
}

func (s *GoFileServer) upload(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	file, header, _ := r.FormFile("file")
	defer func() {
		file.Close()
		r.MultipartForm.RemoveAll()
	}()
	filename := r.FormValue("filename")
	if filename == "" {
		filename = header.Filename
	}
	dst, _ := os.Create(filepath.Join(s.Root, path, filename))
	defer dst.Close()
	n, _ := io.Copy(dst, file)
	log.Println("copy", n, "bytes successfully")
}

func getContent(name string) string {
	f, _ := Assets.Open(name)
	content, _ := ioutil.ReadAll(f)
	return string(content)
}

func renderHTML(w http.ResponseWriter, name string, v interface{}) {
	var funcMap = make(map[string]interface{})
	t := template.Must(template.New(name).Funcs(funcMap).Delims("[[", "]]").Parse(getContent(name)))
	err := t.Execute(w, v)
	if err != nil {
		log.Println(err)
	}
}

func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Println(path, "does not exist")
		return false
	}
	return fileInfo.IsDir()
}