package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"time"

	"github.com/nfnt/resize"
	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"
	"github.com/tidusant/c3m-common/mystring"

	"github.com/tidusant/c3m-common/mycrypto"
	"github.com/tidusant/chadmin-repo/models"
	rpsex "github.com/tidusant/chadmin-repo/session"
	rpimg "github.com/tidusant/chadmin-repo/vrsgim"

	//"io"

	"net/http"
	"net/rpc"

	//	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func init() {

}

type RestResponse struct {
	status  int
	error   string
	message string
	data    json.RawMessage
}

func main() {
	var port int
	var debug bool
	//fmt.Println(mycrypto.Encode("abc,efc", 5))
	flag.IntVar(&port, "port", 8082, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")

	flag.Parse()

	logLevel := log.DebugLevel
	if !debug {
		logLevel = log.InfoLevel
		gin.SetMode(gin.ReleaseMode)
	}

	log.SetOutputFile(fmt.Sprintf("portal-"+strconv.Itoa(port)), logLevel)
	defer log.CloseOutputFile()
	log.RedirectStdOut()

	log.Infof("running with port:" + strconv.Itoa(port))

	//init config

	router := gin.Default()

	router.GET("/*name", func(c *gin.Context) {
		strrt := c3mcommon.Fake64()
		requestDomain := c.Request.Header.Get("Origin")
		if requestDomain == "" {
			requestDomain = c.Request.Host
		}
		allowDomain := c3mcommon.CheckDomain(requestDomain)
		c.Header("Access-Control-Allow-Origin", "*")
		if allowDomain != "" {
			c.Header("Access-Control-Allow-Origin", requestDomain)
			c.Header("Access-Control-Allow-Headers", "access-control-allow-origin, access-control-allow-headers,access-control-allow-credentials")
			c.Header("Access-Control-Allow-Credentials", "true")
			if rpsex.CheckRequest(c.Request.URL.Path, c.Request.UserAgent(), c.Request.Referer(), c.Request.RemoteAddr, "GET") {
				myRouteGET(c)
			} else {
				log.Debugf("check request error")
				c.String(http.StatusOK, strrt)
			}

		} else {
			log.Debugf("Not allow " + requestDomain)
			c.String(http.StatusOK, strrt)
		}

	})

	router.POST("/*name", func(c *gin.Context) {
		strrt := c3mcommon.Fake64()
		requestDomain := c.Request.Header.Get("Origin")
		allowDomain := c3mcommon.CheckDomain(requestDomain)
		c.Header("Access-Control-Allow-Origin", "*")
		if allowDomain != "" {
			c.Header("Access-Control-Allow-Origin", requestDomain)
			c.Header("Access-Control-Allow-Headers", "access-control-allow-origin, access-control-allow-headers,access-control-allow-credentials")
			c.Header("Access-Control-Allow-Credentials", "true")
			log.Debugf("check request:%s", c.Request.URL.Path)
			if rpsex.CheckRequest(c.Request.URL.Path, c.Request.UserAgent(), c.Request.Referer(), c.Request.RemoteAddr, "POST") {
				//c.Body = http.MaxBytesReader(w, c.Body, MaxFileSize)
				// MaxFileSize := 500 * 1024
				// c.Request.ParseMultipartForm(int64(MaxFileSize))

				rs := myRoute(c)
				b, _ := json.Marshal(rs)
				strrt = string(b)
				strrt = mycrypto.Encode(strrt, 8)
				c.String(http.StatusOK, strrt)
			} else {
				log.Debugf("check request error")
				c.String(http.StatusOK, strrt)
			}

		} else {
			log.Debugf("Not allow " + requestDomain)
			c.String(http.StatusOK, strrt)
		}

	})

	router.OPTIONS("/*name", func(c *gin.Context) {
		strrt := c3mcommon.Fake64()
		requestDomain := c.Request.Header.Get("Origin")
		allowDomain := c3mcommon.CheckDomain(requestDomain)
		c.Header("Access-Control-Allow-Origin", "*")
		if allowDomain != "" {
			c.Header("Access-Control-Allow-Origin", requestDomain)
			c.Header("Access-Control-Allow-Headers", "access-control-allow-origin, access-control-allow-headers,access-control-allow-credentials")
			c.Header("Access-Control-Allow-Credentials", "true")
			log.Debugf("check request:%s", c.Request.URL.Path)
			if rpsex.CheckRequest(c.Request.URL.Path, c.Request.UserAgent(), c.Request.Referer(), c.Request.RemoteAddr, "OPTIONS") {
				//c.Body = http.MaxBytesReader(w, c.Body, MaxFileSize)
				// MaxFileSize := 500 * 1024
				// c.Request.ParseMultipartForm(int64(MaxFileSize))
				c.String(http.StatusOK, strrt)
			} else {
				log.Debugf("check request error")
				c.String(http.StatusOK, strrt)
			}

		} else {
			log.Debugf("Not allow " + requestDomain)
			c.String(http.StatusOK, strrt)
		}

	})

	router.Run(":" + strconv.Itoa(port))

}

func myRouteGET(c *gin.Context) {
	strrt := c3mcommon.Fake64()
	mycookie, _ := c.Request.Cookie("sex")
	log.Debugf("get cookie myc %s", mycookie)
	// ck := mycookie.Value
	name := c.Param("name")
	name = name[1:] //remove slash
	requesturl := mycrypto.Decode(name)

	urls := strings.Split(requesturl, "|")
	session := urls[0]
	reqtype := ""
	if len(urls) > 1 {
		reqtype = urls[1]
	}
	if session == "" {
		c.String(http.StatusOK, strrt)
		return
	}
	//check auth
	request := "aut|" + mycrypto.Decode(session)
	rs := c3mcommon.RequestMainService(request, "POST", "aut")
	if rs.Status != "1" {
		c.String(http.StatusOK, strrt)
		return
	}
	logininfo := ""
	json.Unmarshal([]byte(rs.Data), &logininfo)
	shopargs := strings.Split(logininfo, "[+]")

	userid := shopargs[0]
	shopid := ""
	if len(shopargs) > 1 {
		shopid = shopargs[1]
	}
	if userid == "" || shopid == "" {
		c.String(http.StatusOK, strrt)
		return
	}
	//check request type
	if reqtype == "image" {
		filename := "noimage"
		if len(urls) > 2 {
			filename = urls[2]
		}
		fileserve := viper.GetString("config.imagefolder") + shopid + "/" + filename
		http.ServeFile(c.Writer, c.Request, fileserve)
		return
	}

	c.String(http.StatusOK, strrt)

}

func myRoute(c *gin.Context) models.RequestResult {

	name := c.Param("name")
	name = name[1:] //remove slash
	requesturl := mycrypto.Decode(name)
	// mycookie, _ := c.Request.Cookie("sex")
	cks := c.Request.Cookies()

	log.Debugf("get cookie myc %d %v ", len(cks), cks)
	urls := strings.Split(requesturl, "|")
	session := urls[0]

	if session == "" {
		return c3mcommon.ReturnJsonMessage("-2", "session not found", "", "")

	}
	//check auth
	request := "aut|" + session
	rs := c3mcommon.RequestMainService(request, "POST", "aut")
	if rs.Status != "1" {
		return rs
	}
	logininfo := ""
	json.Unmarshal([]byte(rs.Data), &logininfo)
	shopargs := strings.Split(logininfo, "[+]")

	userid := shopargs[0]
	shopid := ""
	if len(shopargs) > 1 {
		shopid = shopargs[1]
	}
	if userid == "" || shopid == "" {
		return c3mcommon.ReturnJsonMessage("-3", "not authorize", "", "")
	}

	data := c.PostForm("data")
	data = mycrypto.Decode(data)
	args := strings.Split(data, "|")
	RPCname := args[0]
	data = data[len(RPCname)+1:]

	if RPCname == "img" && args[1] == "ul" {
		return doUpload(session, userid, shopid, c)
	} else {
		reply := models.RequestResult{}
		client, err := rpc.Dial("tcp", viper.GetString("RPCname."+RPCname))
		if c3mcommon.CheckError("dial RPC"+RPCname+"."+data, err) {
			rpcCall := client.Go("Arith.Run", session+"|"+logininfo+"|"+data, &reply, nil)
			rpcreplyCall := <-rpcCall.Done
			c3mcommon.CheckError("RPC"+RPCname+"."+data, rpcreplyCall.Error)
			client.Close()
		} else {
			reply = c3mcommon.ReturnJsonMessage("-1", "service not run", "", "")
		}
		return reply
	}

}

func doUpload(session, userid, shopid string, c *gin.Context) models.RequestResult {

	filename := mycrypto.Decode(c.PostForm("filename"))
	//get config

	// if shop.Config.Level == 0 {
	// 	return c3mcommon.ReturnJsonMessage("0", "config error", "", "")
	// }
	uploadfolder := viper.GetString("config.imagefolder") + shopid

	//check folder exist
	if _, err := os.Stat(uploadfolder); os.IsNotExist(err) {
		//create shop folder
		os.MkdirAll(uploadfolder, 0755)
		//return c3mcommon.ReturnJsonMessage("0", "folder not found", "", "")

	}
	//get file count
	filecount := rpimg.ImageCount(shopid)
	if filecount == -1 {
		return c3mcommon.ReturnJsonMessage("0", "image count error", "", "")
	}
	//get shop limit
	request := "shop|" + session
	rs := c3mcommon.RequestMainService(request, "POST", "lims|"+shopid)
	if rs.Status != "1" {
		return rs
	}
	var limits []models.ShopLimit
	b, _ := json.Marshal(rs)
	log.Debugf("rs:%s", string(b))
	json.Unmarshal([]byte(rs.Data), &limits)

	limitsmap := make(map[string]int)
	for _, limit := range limits {
		limitsmap[limit.Key] = limit.Value
	}
	maximage := 0
	maxsize := 0
	if val, ok := limitsmap["maximage"]; ok {
		maximage = val
	}
	if val, ok := limitsmap["maxsizeupload"]; ok {
		maxsize = val
	}

	//if _, err := os.Stat(uploadfolder); err == nil {
	//	// path/to/whatever exists
	//}

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
	albumid := c.PostForm("tab")

	// if len(form.Value["tab"]) > 0 {
	// 	albumid = form.Value["tab"][0]
	// }

	strrt := "["

	//log.Debugf("maxupload: %d", shop.Config.MaxImage)

	for _, file := range files {

		strrt += `{"Key":"` + filename + `","Status":`
		//check file count
		if filecount >= maximage {
			strrt += `0,"Value":"Max file reach, please upgrade"},`
			continue
		}

		filetmp, _ := file.Open()

		//file name
		timeint := time.Now().Unix()
		filename := fmt.Sprintf("%d", timeint) + "_" + mystring.RandString(4)

		//check filetype
		buff := make([]byte, 512) // docs tell that it take only first 512 bytes into consideration
		if _, err := filetmp.Read(buff); err != nil {
			c3mcommon.CheckError("error reading file", err)
			strrt += `0,"Value":"Invalid image file"},`
			continue
		}

		//imginf := myimage.GetFormat()
		//imginf,_,_ := image.DecodeConfig(filetmp)
		filetype := http.DetectContentType(buff)
		if filetype != "image/jpeg" && filetype != "image/png" && filetype != "image/gif" {
			strrt += `0,"Value":"Image accept: jpeg,png,gif"},`
			continue
		}
		//check filesize
		filesize, _ := filetmp.Seek(0, 2)
		filetmp.Seek(0, 0)
		if filesize > int64(maxsize*1000*1024) {
			strrt += `0,"Value":"File is larger ` + strconv.Itoa(maxsize) + `MB"},`
			continue
		}
		//save thumb
		imagecontent, _, err := image.Decode(filetmp)
		if !c3mcommon.CheckError("error upload", err) {
			strrt += `0,"Value":"Cannot decode image"},`
			continue
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

		strrt += `1,"Value":"` + filename + `"},`

		//save to db

		rpimg.SaveImage(models.CHImage{Uid: userid, Shopid: shopid, AlbumID: albumid, AppName: viper.GetString("config.appname"), Filename: filename, Created: timeint})

		filecount++
	}
	if len(files) > 0 {
		strrt = strrt[:len(strrt)-1]
	}
	strrt += "]"
	log.Debugf("upload return:%s", strrt)
	return c3mcommon.ReturnJsonMessage("1", "", "", strrt)

}
