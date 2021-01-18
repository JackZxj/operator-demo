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
	"time"
)

// shows the url
var target = ""
var sourceIP = ""
var sourceFile = ""
var done = false

// FileSum shows filename and it's md5
type FileSum struct {
	MD5 string `json:"md5"`
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
func main() {
	target = os.Getenv("DESTINATION")
	sourceIP = os.Getenv("SOURCE_IP")
	source := os.Getenv("SOURCE")
	sourceFile = filepath.Base(source)
	if target == "" || sourceIP == "" || sourceFile == "" {
		log.Fatal("Can't get two env")
	}
	log.Println("I Got file info from operator:\nsourceIP:", sourceIP,
		"sourceFile:", sourceFile, "target:", target)

	go func() {
		for {
			err := fileExists(target)
			url := "http://" + sourceIP + "/" + sourceFile
			if err == nil {
				log.Println("Target file is ready. Check it's md5")
				resp, err := http.Get("http://" + sourceIP + "/md5/" + sourceFile)
				if err != nil {
					log.Fatal("Can't get md5", err)
				}
				body, err := ioutil.ReadAll(resp.Body)
				var sourceFileSum FileSum
				err = json.Unmarshal(body, &sourceFileSum)
				if err != nil {
					log.Fatal(err)
				}
				contents, err := ioutil.ReadFile(target)
				if err != nil {
					log.Fatal("Can't read the target file:", err)
				}
				localMD5Sum := fmt.Sprintf("%x", md5.Sum(contents))
				if localMD5Sum != sourceFileSum.MD5 {
					done = false
					log.Println("local md5:", localMD5Sum, "\tremote md5:", sourceFileSum.MD5)
					log.Println("download target file from remote")
					err = downloadFile(url, target)
					if err != nil {
						log.Panicln(err)
					}
				} else {
					log.Println("Great, the target file has sync!")
					done = true
					time.Sleep(time.Second * 25)
				}
			} else {
				log.Println("Target file is not exist!")
				path, err := filepath.Abs(filepath.Dir(target))
				if err != nil {
					log.Panicln(err)
				}
				err = os.MkdirAll(path, 0777)
				if err != nil {
					log.Panicln(err)
				}
				err = downloadFile(url, target)
				if err != nil {
					log.Panicln(err)
				}
			}
			time.Sleep(time.Second * 5)
		}
	}()

	http.HandleFunc("/", fileStatusHander)
	if err := http.ListenAndServe("0.0.0.0:8081", nil); err != nil {
		log.Fatal("Can't start server:", err)
	}
}

// check ready
func fileStatusHander(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/json")
	info := map[string]bool{"isReady": done}
	msg, err := json.Marshal(info)
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

// downloadFile download target file from remote
func downloadFile(url string, filePath string) error {
	var buf = make([]byte, 32*1024)

	tmpFilePath := filePath + ".download"
	log.Println("tmpFilePath: ", tmpFilePath)

	client := new(http.Client)
	client.Timeout = time.Second * 60
	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	if fileExists(tmpFilePath) == nil {
		log.Println("Remove exist tmpFilePath")
		err = os.Remove(tmpFilePath)
		if err != nil {
			return err
		}
	}
	file, err := os.Create(tmpFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	if resp.Body == nil {
		return errors.New("body is null")
	}
	defer resp.Body.Close()
	//下面是 io.copyBuffer() 的简化版本
	for {
		//读取bytes
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			//写入bytes
			nw, ew := file.Write(buf[0:nr])
			//写入出错
			if ew != nil {
				err = ew
				break
			}
			//读取是数据长度不等于写入的数据长度
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	if err == nil {
		file.Close()
		log.Println("remove local file, and resync from remote")
		err = os.Remove(filePath)
		if os.IsExist(err) {
			return err
		}
		err = os.Rename(tmpFilePath, filePath)
	}
	return err
}
