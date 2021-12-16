package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DrmagicE/gmqtt/config"
	_ "github.com/DrmagicE/gmqtt/persistence"
	"github.com/DrmagicE/gmqtt/server"
	_ "github.com/DrmagicE/gmqtt/topicalias/fifo"
	ctrl "github.com/crosstyan/mqtt-to-ws/controller"
	docs "github.com/crosstyan/mqtt-to-ws/docs"
	l "github.com/crosstyan/mqtt-to-ws/logger"
	"github.com/crosstyan/mqtt-to-ws/model"
	"github.com/crosstyan/mqtt-to-ws/utils"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/pborman/getopt"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// https://stackoverflow.com/questions/1714236/getopt-like-behavior-in-go
var logger = l.Lsugar

var (
	mqttToWs = make(chan model.MQTTMsg)
	mqttToDB = make(chan model.MQTTMsg)
)

// TODO: Maybe I should use a standalone subscription by MQTT client instead of using hooks
// gMQTT hooks for imcoming MQTT Message
var onMsgArrived server.OnMsgArrived = func(ctx context.Context, client server.Client, req *server.MsgArrivedRequest) error {
	// spew.Dump(req)
	// TODO: Add client ID to identify which device is sending the message
	mqttMsg := model.MQTTMsg{
		Topic:   string(req.Publish.TopicName),
		Payload: string(req.Publish.Payload),
	}
	mqttToWs <- mqttMsg
	mqttToDB <- mqttMsg
	return nil
}

var hooks = server.Hooks{
	OnMsgArrived: onMsgArrived,
}

// the swagger package used is https://github.com/swaggo/swag
// instead of https://github.com/go-swagger/go-swagger
// @title           Swagger Example API
// @version         0.1
// @description     This is a sample server celler server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Crosstyan
// @contact.url    https://github.com/crosstyan/mqtt-to-ws/
// @contact.email  crosstyan@outlook.com

// @license.name  WTFPL
// @license.url   http://www.wtfpl.net/

// @host      localhost:8080
// @BasePath  /
func main() {
	// addrLocal, _ := net.InterfaceAddrs()
	// logger.Infof("Local IP: %v", addrLocal)
	var addrHTTP = getopt.StringLong("addr-http", 'a', "0.0.0.0:8080", "HTTP API address", "addr:port")
	var addrMQTT = getopt.StringLong("addr-mqtt", 'A', "0.0.0.0:1883", "MQTT broker address", "addr:port")
	var addrSwagger = getopt.StringLong("addr-swagger", 's', "localhost:8080",
		"Swagger BaseURL -- change this if swagger is not working correctly",
		"addr:port")
	var mongoDBURL = getopt.StringLong("mongo-url", 'M', "mongodb://127.0.0.1:27017/",
		"MongoDB connection URL\nmongodb://[username:password@]host1[:port1][,...hostN[:portN]][/[defaultauthdb][?options]]",
		"url")
	var databaseName = getopt.StringLong("database", 'D', "mqtt", "Database name", "database")
	var websocketPath = getopt.StringLong("websocket", 'w', "/ws", "Websocket listening path -- default '/ws'", "path")
	getopt.Parse()
	ln, err := net.Listen("tcp", *addrMQTT)
	if err != nil {
		logger.Fatal(err.Error())
		return
	}
	docs.SwaggerInfo.Host = *addrSwagger
	db, err := model.GetDB(*mongoDBURL, *databaseName)
	// https://stackoverflow.com/questions/42770022/should-err-error-be-used-in-string-formatting
	if err != nil {
		logger.Fatal(err.Error())
		return
	}

	// gMQTT server
	s := server.New(
		server.WithTCPListener(ln),
		server.WithHook(hooks),
		server.WithLogger(l.L),
		server.WithConfig(config.DefaultConfig()),
	)

	// handle MongoDB message
	go model.HandleMQTTtoDB(mqttToDB, db)

	// start gin server
	go func() {
		hub := utils.NewWsHub(mqttToWs)
		go hub.Run()
		r := gin.New()
		// Config zap logger for gin
		r.Use(ginzap.Ginzap(l.L, time.RFC3339, true))
		r.Use(ginzap.RecoveryWithZap(l.L, true))
		// WebSocket Path
		r.GET(*websocketPath, func(c *gin.Context) {
			utils.ServeWs(hub, c.Writer, c.Request)
		})

		r.GET("/temperature",
			func(c *gin.Context) {
				ctrl.HandleQueryByPage(c, "temperature", db)
			})
		r.GET("/humidity", func(c *gin.Context) {
			ctrl.HandleQueryByPage(c, "humidity", db)
		})
		r.POST("/temperature", func(c *gin.Context) {
			ctrl.HandleQuery(c, "temperature", db)
		})
		r.POST("/humidity", func(c *gin.Context) {
			ctrl.HandleQuery(c, "humidity", db)
		})
		// Swagger in Gin
		// hostname:port/swagger/index.html
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		r.Run(*addrHTTP)
	}()

	// Waiting for stop signal from OS
	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		<-signalCh
		s.Stop(context.Background())
	}()
	// start gMQTT server in main goroutine
	err = s.Run()
	if err != nil {
		panic(err)
	}
}
