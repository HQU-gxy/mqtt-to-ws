package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/DrmagicE/gmqtt/config"
	_ "github.com/DrmagicE/gmqtt/persistence"
	"github.com/DrmagicE/gmqtt/server"
	_ "github.com/DrmagicE/gmqtt/topicalias/fifo"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const (
	dbHost = "127.0.0.1"
	dbPort = "27017"
	dbName = "mqtt"
)

// https://stackoverflow.com/questions/1714236/getopt-like-behavior-in-go
var addr = flag.String("addr", ":8080", "http service address")
var l, _ = zap.NewDevelopment()
var lsugar = l.Sugar()
var (
	mqttToWs = make(chan MQTTMsg)
	mqttToDb = make(chan MQTTMsg)
)

var onMsgArrived server.OnMsgArrived = func(ctx context.Context, client server.Client, req *server.MsgArrivedRequest) error {
	// spew.Dump(req)
	mqttMsg := MQTTMsg{
		Topic:   string(req.Publish.TopicName),
		Payload: string(req.Publish.Payload),
	}
	mqttToWs <- mqttMsg
	mqttToDb <- mqttMsg
	return nil
}

func handleQueryByPage(c *gin.Context, collection string, db *mongo.Database) {
	pageUnparsed := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageUnparsed)
	if err != nil {
		lsugar.Error(err.Error())
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	records, err := GetRecordsByPage(db, collection, int64(page))
	if err != nil {
		lsugar.Error(err.Error())
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	if records == nil {
		// a trick to return empty array
		// https://stackoverflow.com/questions/56200925/return-an-empty-array-instead-of-null-with-golang-for-json-return-with-gin
		var empty = make([]string, 0)
		c.JSON(200, gin.H{"records": empty})
		return
	}
	// RFC3339 is the format of "timestamp"
	c.JSON(200, gin.H{"records": records})
}

var hooks = server.Hooks{
	OnMsgArrived: onMsgArrived,
}

func main() {
	ln, err := net.Listen("tcp", ":1883")
	if err != nil {
		lsugar.Fatal(err.Error())
		return
	}
	connectionURI := "mongodb://" + dbHost + ":" + dbPort + "/"
	db, err := GetDB(connectionURI, dbName)
	// https://stackoverflow.com/questions/42770022/should-err-error-be-used-in-string-formatting
	if err != nil {
		lsugar.Fatal(err.Error())
		return
	}

	// gMQTT server
	s := server.New(
		server.WithTCPListener(ln),
		server.WithHook(hooks),
		server.WithLogger(l),
		server.WithConfig(config.DefaultConfig()),
	)

	// handle MongoDB message
	go func() {
		for {
			msg := <-mqttToDb
			switch msg.Topic {
			case "temperature":
				val, err := msg.ToRecord()
				if err != nil {
					lsugar.Error(err)
					break
					// Prevent the execution of the following code
				}
				CreateRecord(db, "temperature", val)
			case "humidity":
				val, err := msg.ToRecord()
				if err != nil {
					lsugar.Error(err)
					break
				}
				CreateRecord(db, "humidity", val)
			default:
				// ignore
			}
		}
	}()

	// I have totally no idea what this is for
	// 等待中断信号以优雅地关闭服务器
	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		<-signalCh
		s.Stop(context.Background())
	}()

	// gin server
	go func() {
		flag.Parse()
		hub := newHub(mqttToWs)
		go hub.run()
		// r := gin.Default()
		r := gin.New()
		r.Use(ginzap.Ginzap(l, time.RFC3339, true))
		r.Use(ginzap.RecoveryWithZap(l, true))
		r.GET("/ws", func(c *gin.Context) {
			serveWs(hub, c.Writer, c.Request)
		})
		r.GET("/temperature", func(c *gin.Context) {
			handleQueryByPage(c, "temperature", db)
		})
		r.GET("/humidity", func(c *gin.Context) {
			handleQueryByPage(c, "humidity", db)
		})
		r.Run(*addr)
	}()

	// gMQTT server
	err = s.Run()
	if err != nil {
		panic(err)
	}
}
