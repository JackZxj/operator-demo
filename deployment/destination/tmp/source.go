package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// info shows the target file
var info = ""

// FileSum shows filename and it's md5
type FileSum struct {
	MD5 string `json:"md5"`
}

var targetMD5 FileSum

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	info = os.Getenv("TARGET")
	if info == "" {
		log.Fatal("Can't get target env")
	}
	log.Println("I Got file path from operator:", info)
	err := fileExists(info)
	if err != nil {
		log.Fatal("File may not exist! \n", err)
	}
	contents, err := ioutil.ReadFile(info)
	if err != nil {
		log.Fatal("Can't read the target file:", err)
	}
	targetMD5.MD5 = fmt.Sprintf("%x", md5.Sum(contents))
	log.Println(filepath.Base(info), targetMD5.MD5)

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/"+filepath.Base(info), fileHandler)
	http.HandleFunc("/md5/"+filepath.Base(info), checkSumHander)

	if err = http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		log.Fatal("Can't start server:", err)
	}
}

// indexHandler return target file in web
func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Target file: "+info)
}

// file downloader
func fileHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Request from", r.RemoteAddr)
	file, err := os.Open(info)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Panic(err)
		return
	}
	defer file.Close()

	fileHeader := make([]byte, 512)
	file.Read(fileHeader)

	fileStat, _ := file.Stat()

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(info))
	w.Header().Set("Content-Type", http.DetectContentType(fileHeader))
	w.Header().Set("Content-Length", strconv.FormatInt(fileStat.Size(), 10))

	file.Seek(0, 0)
	io.Copy(w, file)
}

// checkSum
func checkSumHander(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/json")
	msg, err := json.Marshal(targetMD5)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Panic(err)
		return
	}
	w.Write(msg)
}

// fileExists check the target file
func fileExists(path string) error {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if os.IsNotExist(err) {
		return err
	}
	return nil
}
