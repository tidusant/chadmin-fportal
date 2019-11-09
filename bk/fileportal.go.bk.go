package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"

	"github.com/tidusant/c3m-common/mycrypto"
	"github.com/tidusant/chadmin-repo/models"
	rpsex "github.com/tidusant/chadmin-repo/session"

	//"io"
	"net"
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

func main() {
	var port int
	var debug bool
	//fmt.Println(mycrypto.Encode("abc,efc", 5))
	flag.IntVar(&port, "port", 8081, "help message for flagname")
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

	router.POST("/*name", func(c *gin.Context) {
		//Origin: domain website call, empty is program call
		//Host: domain from request call. ex: request x.local => host:x.local,
		//request localhost.com:8081 =>host:localhost.com:8081
		//request 192.168.1.223:8081 =>host:192.168.1.223:8081
		//RemoteAddr: userip call.
		//RemoteAddr: ipcall
		requestDomain := c.Request.Header.Get("Origin")

		allowDomain := c3mcommon.CheckDomain(requestDomain)

		strrt := ""
		c.Header("Access-Control-Allow-Origin", "*")

		if allowDomain != "" {
			c.Header("Access-Control-Allow-Origin", requestDomain)
			c.Header("Access-Control-Allow-Headers", "access-control-allow-origin, access-control-allow-headers,access-control-allow-credentials")
			c.Header("Access-Control-Allow-Credentials", "true")
			log.Debugf("check request:%s", c.Request.URL.Path)
			if rpsex.CheckRequest(c.Request.URL.Path, c.Request.UserAgent(), c.Request.Referer(), c.Request.RemoteAddr, "POST") {
				rs := myRoute(c)
				b, _ := json.Marshal(rs)

				strrt = string(b)
			} else {
				log.Debugf("check request error")
			}
		} else {
			log.Debugf("Not allow " + requestDomain)
		}
		if strrt == "" {
			strrt = c3mcommon.Fake64()
		} else {

			strrt = mycrypto.Encode(strrt, 8)
		}
		c.String(http.StatusOK, strrt)

	})

	router.Run(":" + strconv.Itoa(port))

}

func myRoute(c *gin.Context) models.RequestResult {

	name := c.Param("name")
	name = name[1:] //remove  slash
	data := c.PostForm("data")
	// data = c.Request.GetBody().PostForm("data")
	log.Debugf("header:%v", c.Request.Header)
	log.Debugf("Request:%v", c.Request)
	userIP, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	log.Debugf("decode name:%s", mycrypto.Decode(name))
	args := strings.Split(mycrypto.Decode(name), "|")
	data = mycrypto.Decode(data)
	datargs := strings.Split(data, "|")
	RPCname := args[0]

	if RPCname == "CreateSex" {
		data = rpsex.CreateSession()
		b, _ := json.Marshal(data)
		return c3mcommon.ReturnJsonMessage("1", "", "", string(b))

	}

	session := ""
	if len(args) > 1 {
		session = args[1]
	}

	//get session from other server's call
	if session == "" && datargs[0] == "test" && len(datargs) > 1 {
		session = mycrypto.Decode(datargs[1])
	}

	reply := c3mcommon.ReturnJsonMessage("0", "unknown error", "", "")

	//check session
	if !rpsex.CheckSession(session) {
		return c3mcommon.ReturnJsonMessage("-2", "session not found", "", "")
	}
	if RPCname == "aut" {
		//check rpc  is running
		client, err := rpc.Dial("tcp", viper.GetString("RPCname.aut"))
		if err != nil {
			return c3mcommon.ReturnJsonMessage("-1", "service not run", "", "")
		}
		rpcCall := client.Go("Arith.Run", session+"|"+userIP+"|"+data, &reply, nil)
		rpcreplyCall := <-rpcCall.Done
		if rpcreplyCall.Error != nil {
			client.Close()
			return c3mcommon.ReturnJsonMessage("-1", rpcreplyCall.Error.Error(), "", "")
		}
		client.Close()
		log.Debugf("uselogintest %s:", string(reply.Data))
		return reply

	}

	//check login
	client, err := rpc.Dial("tcp", viper.GetString("RPCname.aut"))
	if err != nil {
		return c3mcommon.ReturnJsonMessage("-1", "service not run", "", "")
	}
	rpcCall := client.Go("Arith.Run", session+"|"+userIP+"|"+"aut", &reply, nil)
	rpcreplyCall := <-rpcCall.Done
	if rpcreplyCall.Error != nil {
		client.Close()
		return c3mcommon.ReturnJsonMessage("-1", rpcreplyCall.Error.Error(), "", "")
	}
	client.Close()
	if reply.Status != "1" {
		return reply
	}
	//get logininfo:
	var logininfo string
	json.Unmarshal([]byte(reply.Data), &logininfo)
	//RPC call

	// Synchronous call
	//								log.Debugf("data:%s", mycrypto.Decode(data))
	//								client, err := rpc.Dial("tcp", ":"+viper.GetString("RPCname.Auth"))
	//								c3mcommon.CheckError("dial RPCAuth", err)
	//								err = client.Call("Arith.Run", mycrypto.Decode(data), &reply)
	//								client.Close()
	//								c3mcommon.CheckError("RPCAuth.Call", err)

	//Asyn call only for http
	//check rpc running
	reply = c3mcommon.ReturnJsonMessage("0", "unknown error", "", "")
	client, err = rpc.Dial("tcp", viper.GetString("RPCname."+RPCname))
	if err != nil {
		return c3mcommon.ReturnJsonMessage("-1", "service not run", "", "")
	}
	rpcCall = client.Go("Arith.Run", session+"|"+logininfo+"|"+data, &reply, nil)
	rpcreplyCall = <-rpcCall.Done
	if rpcreplyCall.Error != nil {
		client.Close()
		return c3mcommon.ReturnJsonMessage("-1", rpcreplyCall.Error.Error(), "", "")
	}
	client.Close()
	return reply
}
