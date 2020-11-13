package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"text/template"
	//"io/ioutil"

	"gopkg.in/alecthomas/kingpin.v2"
	//"gopkg.in/yaml.v2"
)

type Configure struct {
	Root			string	 `yaml:"root"`
	Addr 			string 	 `yaml:"addr"`
	Port 			int 	 `yaml:"port"`
	Upload          bool     `yaml:"upload"`
	Delete          bool     `yaml:"delete"`
}

var (
    Assets http.FileSystem = http.Dir("./assets")
	cfg = Configure{}

	VERSION 	= "0.0.1"
	AUTHOR 		= "huangxinyu"
)



func parseFlags() {
	cfg.Root = "./test_data"

	kingpin.HelpFlag.Short('h')
	kingpin.Version(versionMessage())
	kingpin.Flag("port", "listen port, default value is 8000").Short('p').Default("8000").IntVar(&cfg.Port)
	kingpin.Flag("addr", "listen address, eg. 127.0.0.1:8000").Short('a').StringVar(&cfg.Addr)
	kingpin.Flag("upload", "enable upload support").Default("false").Short('u').BoolVar(&cfg.Upload)
	kingpin.Flag("delete", "enable delete support").Default("false").Short('d').BoolVar(&cfg.Delete)
	kingpin.Parse()
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
	parseFlags()
	log.Printf("Configure: %+v", cfg)

	gfs := NewGoFileServer(cfg.Root)
	gfs.Upload = cfg.Upload
	gfs.Delete = cfg.Delete

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
	err := http.ListenAndServe(cfg.Addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}