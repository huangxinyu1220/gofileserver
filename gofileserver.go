package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type GoFileServer struct {
	Root            string
	Upload          bool
	Delete          bool
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

func (s *GoFileServer) status(w http.ResponseWriter, r *http.Request) {
	data, _ := json.MarshalIndent(s, "", "    ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *GoFileServer) mkdir(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	name := r.FormValue("name")
	err := os.Mkdir(filepath.Join(s.Root, path, name), 0755)
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
		renderHTML(w, "index.html", s)
	} else {
		if r.FormValue("download") == "true" {
			w.Header().Set("Content-Disposition", "attachment; filename=" + strconv.Quote(filepath.Base(path)))
		}
		http.ServeFile(w, r, realPath)
	}
}

func (s *GoFileServer) jsonList(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
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
	fileList := make([]FileInfo, 0)
	for path, info := range filePathMap {
		fileInfo := FileInfo {
			Name:    info.Name(),
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano() / 1e6,
			IsDir:   info.IsDir(),
		}
		fileList = append(fileList, fileInfo)
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
	log.Println("get upload request")
	path := mux.Vars(r)["path"]
	file, header, _ := r.FormFile("file")
	log.Println(header.Filename)
	defer func() {
		file.Close()
		log.Println(r.MultipartForm)
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