package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
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
	info = os.Getenv("SOURCE")
	natsServer := os.Getenv("NATS_SERVERS")
	if info == "" || natsServer == "" {
		log.Fatal("Can't get target env")
	}
	log.Println("I Got file path from operator:", info)

	nc, err := nats.Connect(natsServer, nats.Name("demo"))
	if err != nil {
		log.Fatal("connect error")
	}
	defer nc.Close()
	subj, _ := TargetFile2Subject(info)
	log.Println(subj)

	go func() {
		for {
			err := fileExists(info)
			if err != nil {
				log.Fatal("File may not exist! \n", err)
			}
			contents, err := ioutil.ReadFile(info)
			if err != nil {
				log.Fatal("Can't read the target file:", err)
			}
			// targetMD5.MD5 = fmt.Sprintf("%x", md5.Sum(contents))
			currentMD5 := fmt.Sprintf("%x", md5.Sum(contents))
			if currentMD5 != targetMD5.MD5 {
				log.Println("Source file has changed! Last MD5: ", targetMD5.MD5, "\tCurrent MD5:", currentMD5)
				targetMD5.MD5 = currentMD5
			} else {
				log.Println("Great, the source file hasn't changed.")
			}
			_, err = nc.Request(subj+".md5", []byte(targetMD5.MD5), 3*time.Second)
			if err != nil {
				if err == nats.ErrTimeout {
					log.Println("No one reply in 3 seconds. May be all target has updated!")
				} else {
					log.Panicln(err)
				}
			} else {
				time.Sleep(time.Second)
				err = nc.Publish(subj, contents)
				if err != nil {
					log.Println(err)
				}
			}
			time.Sleep(time.Second * 5)
		}
	}()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/"+filepath.Base(info), fileHandler)
	http.HandleFunc("/md5/"+filepath.Base(info), checkSumHander)

	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
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

// TargetFile2Subject recieve source hostname and filepath, returns a subject for NATS
func TargetFile2Subject(filePath string) (string, error) {
	if filePath == "" {
		return "", errors.New("Empty hostname or file path")
	}
	md := md5.Sum([]byte(filePath))
	return fmt.Sprintf("%x", md), nil
}
