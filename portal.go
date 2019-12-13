package main

import (
	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"
	"github.com/tidusant/c3m-common/mycrypto"
	"github.com/tidusant/c3m-common/mystring"
	"github.com/tidusant/chadmin-repo/models"
	rpsex "github.com/tidusant/chadmin-repo/session"
	rpimg "github.com/tidusant/chadmin-repo/vrsgim"

	"flag"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/nfnt/resize"
	//"strings"
	//	"encoding/json"
	//	"io/ioutil"

	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {

	var port int
	var debug bool
	//fmt.Println(mycrypto.Encode("abc,efc", 5))
	flag.IntVar(&port, "port", 8082, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")
	flag.Parse()

	//logLevel := log.DebugLevel
	if !debug {
		//logLevel = log.InfoLevel
		gin.SetMode(gin.ReleaseMode)
	}

	// log.SetOutputFile(fmt.Sprintf("upload-"+strconv.Itoa(port)), logLevel)
	// defer log.CloseOutputFile()
	// log.RedirectStdOut()

	log.Infof("running with port:" + strconv.Itoa(port))

	//init config

	router := gin.Default()
	router.POST("/:name/:ck", func(c *gin.Context) {

		requestDomain := c.Request.Header.Get("Origin")
		allowDomain := c3mcommon.CheckDomain(requestDomain)
		strrt := ""
		c.Header("Access-Control-Allow-Origin", "*")
		if allowDomain != "" {
			c.Header("Access-Control-Allow-Origin", allowDomain)
			c.Header("Access-Control-Allow-Headers", "access-control-allow-origin, access-control-allow-headers,access-control-allow-credentials")
			c.Header("Access-Control-Allow-Credentials", "true")
			action := (c.Param("name"))
			session := mycrypto.Decode(c.Param("ck"))
			data := mycrypto.Decode(c.PostForm("data"))
			album := mycrypto.Decode(c.PostForm("tab"))
			log.Debugf("session: %s,data: %s,album name:%s", session, data, album)

			log.Debugf("request:%s", c.Request.URL.Path)
			reply := c3mcommon.ReturnJsonMessage("-5", "unknown error", "", "")

			if rpsex.CheckRequest(c.Request.URL.Path, c.Request.UserAgent(), c.Request.Referer(), c.Request.RemoteAddr, "POST") {

				//check login
				var postdata url.Values

				datastr := "aut|" + session
				rs := c3mcommon.RequestService(mycrypto.Encode3(datastr), postdata)
				log.Debugf("response %s", rs)
				sessioninfo := strings.Split(rs, "[+]")

				params := strings.Split(data, "|")
				if len(sessioninfo) > 1 {
					userid := sessioninfo[0]
					shopid := sessioninfo[1]
					if action == "files" {
						filenames := mycrypto.Decode(c.PostForm("fileuploadnames"))
						reply = doUpload(userid, shopid, filenames, c)
					} else {
						if action == "loadimage" {
							if len(params) > 0 {
								reply = doLoadImage(userid, shopid, params[0])
							}
						} else if action == "deletefiles" {
							if len(params) > 0 {
								reply = doRemoveImage(userid, params[0], shopid)
							}
						}

					}

				} else {
					reply = c3mcommon.ReturnJsonMessage("-3", "not authorize", "", "")

				}

			} else {
				reply = c3mcommon.ReturnJsonMessage("-2", "session not found", "", "")

			}

			if reply != "" {
				strrt = mycrypto.Encode(reply, 8)
			}
		} else {
			log.Debugf("Not allow " + requestDomain)
		}
		if strrt == "" {
			strrt = c3mcommon.Fake64()
		}
		c.String(http.StatusOK, strrt)
	})

	router.Run(":" + strconv.Itoa(port))

}

func doRemoveImage(userid, filenames, shopid string) string {

	//get config

	if shopid == "" {
		return c3mcommon.ReturnJsonMessage("0", "shop not found", "", "")
	}

	uploadfolder := "./images/" + shopid
	log.Debugf("shopid to del:%s, useid:%s", shopid, userid)
	//check folder exist
	if _, err := os.Stat(uploadfolder); os.IsNotExist(err) {
		return c3mcommon.ReturnJsonMessage("0", "folder not found", "", "")

	}

	files := strings.Split(filenames, ",")

	strrt := "{\"\":\"\""
	removeCount := 0
	for i, file := range files {
		if rpimg.RemoveImage(shopid, userid, file) {
			strrt += ",\"" + strconv.Itoa(i) + "\":\"" + file + "\""
			os.Remove(uploadfolder + "/" + file)
			os.Remove(uploadfolder + "/thumb_" + file)
			removeCount++
		}
	}

	strrt += "}"
	if removeCount == 0 {
		return c3mcommon.ReturnJsonMessage("0", "Cannot remove!", "", strrt)
	}
	return c3mcommon.ReturnJsonMessage("1", "", "", strrt)
}
func doLoadImage(userid, shopid, albumname string) string {
	//get config
	if shopid == "" {
		return c3mcommon.ReturnJsonMessage("0", "shop not found", "", "")
	}
	log.Debugf("loadimage userid:%s, shopiid:%s, albumname:%s", userid, shopid, albumname)
	uploadfolder := "./images/" + shopid
	//check folder exist
	if _, err := os.Stat(uploadfolder); os.IsNotExist(err) {
		return c3mcommon.ReturnJsonMessage("0", "folder not found", "", "")
	}
	//loop user directory
	images := rpimg.GetImages(shopid, userid, albumname)

	strrt := "{\"\":\"\""
	for i, fileimage := range images {
		strrt += ",\"" + strconv.Itoa(i) + "\":\"" + fileimage.Filename + "\""
	}

	strrt += "}"
	return c3mcommon.ReturnJsonMessage("1", "", "", strrt)

}
func doUpload(userid, shopid, filenames string, c *gin.Context) string {
	if shopid == "" {
		return c3mcommon.ReturnJsonMessage("0", "shop not found", "", "")
	}
	//get config

	// if shop.Config.Level == 0 {
	// 	return c3mcommon.ReturnJsonMessage("0", "config error", "", "")
	// }
	uploadfolder := "./images/" + shopid

	//check folder exist
	if _, err := os.Stat(uploadfolder); os.IsNotExist(err) {
		return c3mcommon.ReturnJsonMessage("0", "folder not found", "", "")

	}
	//get file count
	filecount := rpimg.ImageCount(shopid)
	if filecount == -1 {
		return c3mcommon.ReturnJsonMessage("0", "image count error", "", "")

	}
	//if _, err := os.Stat(uploadfolder); err == nil {
	//	// path/to/whatever exists
	//}

	log.Debugf("filecount: %d", filecount)
	//// single file
	//	file, _ := c.FormFile("file")
	//	log.Println(file.Filename)
	//	out, err := os.Create("./tmp/" + file.Filename)
	//	c3mcommon.CheckError("error upload", err)
	//	defer out.Close()
	//	filetmp, _ := file.Open()
	//	_, err = io.Copy(out, filetmp)
	//	c3mcommon.CheckError("error upload", err)

	//multi file
	form, _ := c.MultipartForm()
	files := form.File["file"]
	album := ""
	if len(form.Value["tab"]) > 0 {
		album = form.Value["tab"][0]
	}

	log.Debugf("albumn in func:%s", album)
	uploadnames := strings.Split(filenames, ",")
	strrt := "{\"\":1"
	uploadedcount := 0
	//log.Debugf("maxupload: %d", shop.Config.MaxImage)
	var imgfiles []models.CHImage
	for i, file := range files {
		log.Debugf("filename %d:%s - %s, %v", i, uploadnames[i], file.Filename, file.Header)

		//check file type:
		strrt += ",\"" + uploadnames[i] + "\":"
		// if filecount+uploadedcount+1 > shop.Config.MaxImage {
		// 	strrt += "0"
		// 	continue
		// }
		filetmp, _ := file.Open()

		//file name
		timeint := time.Now().Unix()
		filename := fmt.Sprintf("%d", timeint) + "_" + mystring.RandString(4)

		//check filetype
		buff := make([]byte, 512) // docs tell that it take only first 512 bytes into consideration
		if _, err := filetmp.Read(buff); err != nil {
			c3mcommon.CheckError("error reading file", err)
			strrt += "-3"
			continue
		}
		filetype := http.DetectContentType(buff)
		if filetype != "image/jpeg" && filetype != "image/png" && filetype != "image/gif" {
			strrt += "-1"
			continue
		}
		//check filesize
		filesize, _ := filetmp.Seek(0, 2)
		filetmp.Seek(0, 0)
		if filesize > 10*1000*1024 {
			strrt += "-2"
			continue
		}
		//save thumb
		imagecontent, _, err := image.Decode(filetmp)
		if !c3mcommon.CheckError("error upload", err) {
			return c3mcommon.ReturnJsonMessage("0", "cannot create thumb", "", strrt)
		}
		m := resize.Resize(200, 0, imagecontent, resize.NearestNeighbor)
		out, err := os.Create(uploadfolder + "/thumb_" + filename)
		c3mcommon.CheckError("error create thumb", err)
		defer out.Close()
		//save file
		out2, err := os.Create(uploadfolder + "/" + filename)
		c3mcommon.CheckError("error upload", err)
		defer out2.Close()

		if filetype == "image/jpeg" {
			jpeg.Encode(out, m, nil)
			jpeg.Encode(out2, imagecontent, nil)
		} else if filetype == "image/gif" {
			gif.Encode(out, m, nil)
			gif.Encode(out2, imagecontent, nil)
		} else if filetype == "image/png" {
			png.Encode(out, m)
			png.Encode(out2, imagecontent)
		}

		c3mcommon.CheckError("error upload", err)

		strrt += fmt.Sprintf(`"%s"`, filename)

		//save to db
		imgfiles = append(imgfiles, models.CHImage{Uid: userid, Shopid: shopid, Albumname: album, AppName: viper.GetString("config.appname"), Filename: filename, Created: timeint})
		uploadedcount++
	}
	//save to db
	rpimg.SaveImages(imgfiles)
	strrt += "}"
	return c3mcommon.ReturnJsonMessage("1", "", "", strrt)

}
