package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/nats-io/nats.go"
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
	natsServer := os.Getenv("NATS_SERVERS")
	if target == "" || sourceIP == "" || source == "" || natsServer == "" {
		log.Fatal("Can't get two env")
	}
	sourceFile = filepath.Base(source)
	log.Println("I Got file info from operator:\nsourceIP:", sourceIP,
		"sourceFile:", sourceFile, "target:", target)

	nc, err := nats.Connect(natsServer, nats.Name("demo"))
	if err != nil {
		log.Fatal("connect error")
	}
	defer nc.Close()
	subj, _ := TargetFile2Subject(source)
	log.Println(subj)

	go func() {
		for {
			err := fileExists(target)
			localMD5 := ""
			if err == nil {
				log.Println("Target file is ready. Check it's md5")
				contents, err := ioutil.ReadFile(target)
				if err != nil {
					log.Fatal("Can't read the target file:", err)
				}
				localMD5 = fmt.Sprintf("%x", md5.Sum(contents))
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
			}
			// Subscribe
			sub, err := nc.SubscribeSync(subj + ".md5")
			if err != nil {
				log.Fatal(err)
			}
			// Wait for a message
			msg, err := sub.NextMsg(10 * time.Second)
			if err != nil {
				if err == nats.ErrTimeout {
					log.Println("Server did not publish new md5 in 10 seconds. Retry after 3 second.")
					time.Sleep(3 * time.Second)
					continue
				} else {
					log.Fatal(err)
				}
			}
			log.Printf("Reply: %s", msg.Data)
			remoteMD5 := msg.Data
			if string(remoteMD5) != localMD5 {
				done = false
				log.Println("local md5:", localMD5, "\tremote md5:", string(remoteMD5))
				log.Println("download target file from remote")
				nc.Publish(msg.Reply, []byte("I need that file!"))
				err := downloadFileFromNATS(nc, subj, target)
				if err != nil {
					if err == nats.ErrTimeout {
						log.Println(err)
					} else {
						log.Panicln(err)
					}
				}
			} else {
				log.Println("Great, the target file has sync!")
				done = true
				time.Sleep(time.Second * 25)
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

// TargetFile2Subject recieve source hostname and filepath, returns a subject for NATS
func TargetFile2Subject(filePath string) (string, error) {
	if filePath == "" {
		return "", errors.New("Empty hostname or file path")
	}
	md := md5.Sum([]byte(filePath))
	return fmt.Sprintf("%x", md), nil
}

// ByteSliceEqual checks whether two slices equal
func ByteSliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	// 为了实现[]int{} != []int(nil)
	// https://www.jianshu.com/p/80f5f5173fca
	if (a == nil) != (b == nil) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func downloadFileFromNATS(nc *nats.Conn, subj, filePath string) error {
	tmpFilePath := filePath + ".download"
	log.Println("tmpFilePath: ", tmpFilePath)
	if fileExists(tmpFilePath) == nil {
		log.Println("Remove exist tmpFilePath")
		err := os.Remove(tmpFilePath)
		if err != nil {
			return err
		}
	}
	file, err := os.Create(tmpFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	sub, err := nc.SubscribeSync(subj)
	if err != nil {
		return err
	}
	// Wait for a message
	msg, err := sub.NextMsg(5 * time.Second)
	if err != nil {
		return err
	}
	_, err = file.Write(msg.Data)
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
