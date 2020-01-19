package main

import (
	"encoding/json"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

const port = ":10000"
const basePath = "/home/cameron/Pictures/"

type FileItem struct {
	Name string
	Path string
	Size int64
}

func home(writer http.ResponseWriter, req *http.Request) {
	path, _ := url.QueryUnescape(req.FormValue("path"))
	if path == "" {
		path = basePath
	}
	files, err := ioutil.ReadDir(path)

	if err != nil {
		writer.WriteHeader(400)
		fmt.Println(err)
		return
	}

	fileItems := make([]FileItem, 0, 0)
	dirs := make([]FileItem, 0)
	for i := 0; i < len(files); i++ {
		file := files[i]
		if file.IsDir() {
			dirs = append(dirs, FileItem{
				Name: file.Name(),
				Path: filepath.Join(path, file.Name()),
				Size: file.Size(),
			})
		} else {
			fileItems = append(fileItems, FileItem{
				Name: file.Name(),
				Path: filepath.Join(path, file.Name()),
				Size: file.Size(),
			})
		}
	}

	writer.Header().Add("Content-Type", "application/json")
	err = json.NewEncoder(writer).Encode(map[string][]FileItem{
		"dirs":  dirs,
		"files": fileItems,
	})

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
	}

	fmt.Println("Home hit!")
}

func thumbnail(w http.ResponseWriter, req *http.Request) {
	path, _ := url.QueryUnescape(req.FormValue("path"))
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if _, err := os.Stat(path); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	smallestSide, _ := strconv.Atoi(req.FormValue("size"))
	if smallestSide == 0 {
		smallestSide = 100
	}

	if img, err := imaging.Open(path); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	} else {
		var width = 0
		var height = 0
		if img.Bounds().Dx() > img.Bounds().Dy() {
			height = smallestSide
			width = 0
		} else {
			height = 0
			width = smallestSide
		}
		thumb := imaging.Resize(img, width, height, imaging.Lanczos)
		output, err := ioutil.TempFile(filepath.Dir(path), "*.jpg")
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tempFileName := output.Name()
		saveErr := imaging.Save(thumb, tempFileName, imaging.JPEGQuality(85))
		if saveErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(saveErr)
			return
		}

		http.ServeFile(w, req, tempFileName)
		_ = os.Remove(tempFileName)
		fmt.Println(tempFileName, "served")
	}
}

func image(writer http.ResponseWriter, req *http.Request) {
	var path = req.FormValue("path")
	if _, err := os.Stat(path); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Println(err)
		return
	} else if matches, _ := regexp.MatchString("(jpeg|JPEG|jpg|JPG|png|PNG)", path); matches {
		http.ServeFile(writer, req, path)
	} else {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Println(err)
	}
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", home)
	router.Path("/").Queries("path", "{path}").HandlerFunc(home)
	router.Path("/thumbnail").Queries("path", "{path}").HandlerFunc(thumbnail)
	router.Path("/image").Queries("path", "{path}").HandlerFunc(image)

	http.ListenAndServe(port, router)
}
