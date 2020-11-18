package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"text/template"
)

type Configure struct {
	Conf			*os.File `yaml:"-"`
	Root			string	 `yaml:"root"`
	Addr 			string 	 `yaml:"addr"`
	Port 			int 	 `yaml:"port"`
	Upload          bool     `yaml:"upload"`
	Delete          bool     `yaml:"delete"`
	Theme			string	 `yaml:"theme"`
	Title			string	 `yaml:"title"`
}

var (
    Assets http.FileSystem = http.Dir("./assets")
	cfg = Configure{}

	VERSION 	= "0.0.1"
	AUTHOR 		= "huangxinyu"
)

func parseFlags() error {
	cfg.Root = "./"
	cfg.Port = 8000
	cfg.Addr = ""
	cfg.Theme = "blue"
	cfg.Title = "Go File Server"
	kingpin.HelpFlag.Short('h')
	kingpin.Version(versionMessage())
	kingpin.Flag("conf", "config file path, file format: yaml").FileVar(&cfg.Conf)
	kingpin.Flag("port", "listen port, default value is 8000").Short('p').IntVar(&cfg.Port)
	kingpin.Flag("addr", "listen address, eg. 127.0.0.1:8000").Short('a').StringVar(&cfg.Addr)
	kingpin.Flag("upload", "enable upload support").Short('u').BoolVar(&cfg.Upload)
	kingpin.Flag("delete", "enable delete support").Short('d').BoolVar(&cfg.Delete)
	kingpin.Flag("theme", "theme of ui, [blue, green], default is blue").StringVar(&cfg.Theme)
	kingpin.Flag("title", "title of ui, default is 'Go File Server'").StringVar(&cfg.Title)
	kingpin.Parse()
	if cfg.Conf != nil {
		data, err := ioutil.ReadAll(cfg.Conf)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(data, &cfg)
		if err != nil {
			return err
		}
		kingpin.Parse()
	}
	return nil
}

func versionMessage() string {
	t := template.Must(template.New("version").Parse(`GoFileServer
	Version:	{{.Version}}
	Go version:	{{.GoVersion}}
  	OS/Arch:	{{.OSArch}}
	Author:		{{.Author}}`))
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	err := t.Execute(buf, map[string]interface{}{
		"Version":   VERSION,
		"GoVersion": runtime.Version(),
		"OSArch":    runtime.GOOS + "/" + runtime.GOARCH,
		"Author":      AUTHOR,
	})
	if err != nil {
		log.Println("versionMessage gets error: ", err)
		return ""
	}
	return buf.String()
}

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Configure: %+v", cfg)

	gfs := NewGoFileServer(cfg.Root)
	gfs.Upload = cfg.Upload
	gfs.Delete = cfg.Delete
	gfs.Theme = cfg.Theme
	gfs.Title = cfg.Title

	http.Handle("/", gfs)
	http.Handle("/-/assets/", http.StripPrefix("/-/assets/", http.FileServer(Assets)))
	http.HandleFunc("/-/sysinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(map[string]interface{}{
			"version": VERSION,
		})
		w.Write(data)
	})
	if cfg.Addr == "" {
		cfg.Addr = fmt.Sprintf(":%d", cfg.Port)
	}
	err = http.ListenAndServe(cfg.Addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}