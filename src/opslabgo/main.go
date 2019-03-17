package main

//
// @instruction
//		利用go实现的HTTP版的文件上传，上传接口以json方式返回
// @上传方式
// 		通用的http文件上传方式
// @实现原理
//		当前程序接受到上传请求处理并响应,同时利用go的协程同步到配置的其他客户端上
//
// @配置json
//	{
//    "addr":"0.0.0.0:9090",
//    "path":"c:/var/upload/wwww/",
//    "fileNameLength":11,
//    "rysncAddr":[
//        "http://localhost:9091/rsync",
//        "http://localhost:9092/rsync"
//    ]
//	}
// @说明
// 		普通上传方式的文件都会存储在配置文件指定的目录下，如果想在改目录下新建文件夹并存储
// 		到新建文件夹下的可以同http head字段path添加目录(目录名需要BASE64)

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type ServerConfig struct {
	//监听地址和端口
	ADDR string `json:'ADDR'`
	//文件写入路径
	PATH string `json:'PATH'`
	//文件名随机长度
	FILENAMELENGTH int `json:'FILENAMELENGTH'`
	//同步的地址
	RYSNCADDR []string `json:'RYSNCADDR'`
}

var conf = ServerConfig{}

func main() {

	confile := flag.String("conf", "", "the configuration file")
	flag.Parse()
	if *confile == "" {
		fmt.Println("Please specify the configuration file")
		return
	}
	file, _ := os.Open(*confile)
	defer file.Close()
	decoder := json.NewDecoder(file)

	err := decoder.Decode(&conf)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	//@TODO-FORTEST
	//var jsonstr = `{"addr":"0.0.0.0:9090","path":"c:/var/upload/www/","fileNameLength":10,
	//	"rysncAddr":["http://localhost:9091/rsync","http://localhost:9092/rsync"]}`
	//if err := json.Unmarshal([]byte(jsonstr), &conf); err != nil {
	//	panic("ErrorConfig")
	//}

	log.Println("Server is starting:" + conf.ADDR)
	log.Println("Server UploadPath:" + conf.PATH)
	log.Print("Server Rysnc Addr:" + strings.Replace(strings.Trim(fmt.Sprint(conf.RYSNCADDR), "[]"), " ", ",", -1))

	http.HandleFunc("/upload", UploadHandler)
	http.HandleFunc("/rsync", RsyncHandler)
	if err := http.ListenAndServe(conf.ADDR, nil); err != nil {
		fmt.Println("Server starting error")
	}
}

func RandomFile(path string, suffix string) (string, error) {
	if (!IsFileExist(path)) {
		err := os.MkdirAll(path, os.ModePerm)
		return "", err
	}
	for {
		dstFile := path + NewLenChars(conf.FILENAMELENGTH) + suffix
		if (!IsFileExist(dstFile)) {
			return dstFile, nil
		}
	}
}

func IsFileExist(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

func NewLenChars(length int) string {
	if length == 0 {
		return ""
	}
	var chars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	clen := len(chars)
	if clen < 2 || clen > 256 {
		panic("Wrong charset length for NewLenChars()")
	}
	maxrb := 255 - (256 % clen)
	b := make([]byte, length)
	r := make([]byte, length+(length/4))
	i := 0
	for {
		if _, err := rand.Read(r); err != nil {
			panic("Error reading random bytes: " + err.Error())
		}
		for _, rb := range r {
			c := int(rb)
			if c > maxrb {
				continue
			}
			b[i] = chars[c%clen]
			i++
			if i == length {
				return string(b)
			}
		}
	}
}

func getCurrentIP(r http.Request) (string) {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		return r.RemoteAddr
	}
	return ip
}

func RsyncHandler(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("rsyncfile")
	defer file.Close()
	if err != nil {
		log.Println(fmt.Sprintf("%s rsyncfile %s %s ", getCurrentIP(*r), header.Filename, "FormParseError"))
		res := fmt.Sprintf("{'code':'error'}")
		w.Header().Add("Content-Type", "application/json;charset:utf-8;")
		fmt.Fprintf(w, res)
		return
	}
	dstFile := conf.PATH + header.Filename
	if IsFileExist(dstFile) {
		log.Println(fmt.Sprintf("%s rsyncfile %s %s ", getCurrentIP(*r), header.Filename, "FileExists"))
		res := fmt.Sprintf("{'code':'error'}")
		w.Header().Add("Content-Type", "application/json;charset:utf-8;")
		fmt.Fprintf(w, res)
		return
	}

	cur, err := os.Create(dstFile);
	defer cur.Close()
	if err != nil {
		log.Println(fmt.Sprintf("%s rsyncfile %s %s ", getCurrentIP(*r), header.Filename, "CreateError"))
		res := fmt.Sprintf("{'code':'error'}")
		w.Header().Add("Content-Type", "application/json;charset:utf-8;")
		fmt.Fprintf(w, res)
		return
	}

	res := fmt.Sprintf("{'code':'error'}")
	loginfo := ""
	_, erro := io.Copy(cur, file)
	if erro != nil {
		loginfo = fmt.Sprintf("%s rsyncfile %s  %s", getCurrentIP(*r), header.Filename, "WriteError")
	} else {
		loginfo = fmt.Sprintf("%s rsyncfile %s  %s", getCurrentIP(*r), header.Filename, "RysncSuccess")
		res = fmt.Sprintf("{'code':'success'}")

	}
	log.Println(loginfo)
	w.Header().Add("Content-Type", "application/json;charset:utf-8;")
	fmt.Fprintf(w, res)
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// 实现多文件接收
	//上传结果以以json格式返回
	uploadPath := r.Header.Get("Path")
	basePath := conf.PATH

	re2, _ := regexp.Compile("\\.{2,}")
	re3, _ := regexp.Compile("/{2,}")

	if uploadPath != "" {
		if decodeBytes, err := base64.StdEncoding.DecodeString(uploadPath); err == nil {
			ppath := string(decodeBytes)
			ppath = re3.ReplaceAllString(re2.ReplaceAllString(ppath, ""), "/")
			uploadPath = ppath
			basePath += "/" + ppath
		}
	}
	if (!strings.HasSuffix(basePath, "/")) {
		basePath += "/"
	}

	basePath = re3.ReplaceAllString(basePath, "/")

	bastPathLen := len(conf.PATH) - 1
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s := ""
	res := "success"
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if newfile, err := RandomFile(basePath, path.Ext(part.FileName())); err == nil {
			if part.FileName() != "" {
				dst, _ := os.Create(newfile)
				defer dst.Close()
				io.Copy(dst, part)
				newFileName := string([]byte(newfile)[bastPathLen:])
				log.Println(fmt.Sprintf("%s uploadfile [%s][%s] > %s", getCurrentIP(*r),
					part.FormName(), part.FileName(), newfile))
				s += fmt.Sprintf("%s@%s:'%s',", part.FormName(), part.FileName(), newFileName)
				for _, v := range conf.RYSNCADDR {
					go Rsync(v, uploadPath, newfile)
				}
			}
		} else {
			log.Println(fmt.Sprintf("%s uploadfile [%s][%s] CreateDestinationFileError", getCurrentIP(*r),
				part.FormName(), part.FileName(), newfile))
			s += fmt.Sprintf("%s@%s:'%s',", part.FormName(), part.FileName())
			res = "error"
		}
	}
	w.Header().Add("Content-Type", "application/json;charset:utf-8;")
	fmt.Fprintf(w, fmt.Sprintf("{'code':'%s',results:{%s}}", res, strings.Trim(s, ",")))
}

func Rsync(url string, dstPath string, files string) {
	bodyBuffer := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuffer)
	_, fileName := filepath.Split(files)
	fileWriter, _ := bodyWriter.CreateFormFile("rsyncfile", fileName)

	file, _ := os.Open(files)
	defer file.Close()

	io.Copy(fileWriter, file)

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	if req, err := http.NewRequest("POST", url, bodyBuffer); err == nil {
		req.Header.Set("Content-Type", contentType)
		if dstPath != "" {
			req.Header.Set("Path", base64.StdEncoding.EncodeToString([]byte(dstPath)))
		}
		if resp, errsp := http.DefaultClient.Do(req); errsp == nil {
			resp_body, _ := ioutil.ReadAll(resp.Body)
			log.Println(fmt.Sprintf("Clientrsyncfile %s %s ", resp.Status, string(resp_body)))
		} else {
			log.Println(fmt.Sprintf("Clientrsyncfile Error %s %s ", url, files))
		}

	} else {
		log.Println(fmt.Sprintf("Clientrsyncfile Error %s %s ", url, files))
	}

}
