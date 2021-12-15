package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/DrmagicE/gmqtt/config"
	_ "github.com/DrmagicE/gmqtt/persistence"
	"github.com/DrmagicE/gmqtt/server"
	_ "github.com/DrmagicE/gmqtt/topicalias/fifo"
	docs "github.com/crosstyan/mqtt-to-ws/docs"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

type DateRangeRequest struct {
	// Page is from 1 to infinity
	Page *int64 `json:"page,omitempty" example:"1"`
	// Time RFC3339
	Start *string `json:"start" example:"2020-01-01T00:00:00Z" validate:"required"`
	// Time RFC3339
	End *string `json:"end,omitempty" example:"2022-01-01T00:00:00Z"`
}

type ErrorMsg struct {
	// Error message
	Err string `json:"error" example:"error message"`
}

type ResponseMsg struct {
	Records []MQTTRecord `json:"records"`
}

// TempExample godoc
// @Summary      Get Temperature/Humidity Records by Date
// @Description  get Temperature/Humidity by date
// @Tags         MQTTRecords
// @Produce      json
// @Param        data body DateRangeRequest true "Request Body"
// @Success      200  {object}  ResponseMsg
// @Failure      400  {object}  ErrorMsg
// @Failure      500  {object}  ErrorMsg
// @Router       /temperature [post]
// @Router       /humidity [post]
func handleQuery(c *gin.Context, collection string, db *mongo.Database) {
	var dateRange DateRangeRequest
	c.BindJSON(&dateRange)
	var page int64
	if dateRange.Page == nil || *dateRange.Page <= 0 {
		page = 1
	} else {
		page = *dateRange.Page
	}
	var tEnd time.Time
	var tStart time.Time
	tStart, err := time.Parse(time.RFC3339, *dateRange.Start)
	if err != nil {
		lsugar.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: Refactor this part
	if dateRange.End != nil {
		tEnd, err = time.Parse(time.RFC3339, *dateRange.End)
		if err != nil {
			lsugar.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		records, err := GetRecordsBetween(db, collection, tStart, tEnd, page)
		if err != nil {
			lsugar.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if records == nil {
			var empty = make([]string, 0)
			c.JSON(http.StatusOK, gin.H{"records": empty})
			return
		}
		c.JSON(http.StatusOK, gin.H{"records": records})
	} else {
		records, err := GetRecordsFrom(db, collection, tStart, page)
		if err != nil {
			lsugar.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if records == nil {
			var empty = make([]string, 0)
			c.JSON(http.StatusOK, gin.H{"records": empty})
			return
		}
		c.JSON(http.StatusOK, gin.H{"records": records})
	}
}

// @Summary      Get Temperature/Humidity Records by Page
// @Description  get Temperature/Humidity by page
// @Tags         MQTTRecords
// @Produce      json
// @Param        page query int false "From 1 to infinity"
// @Success      200  {object}  ResponseMsg
// @Failure      400  {object}  ErrorMsg
// @Failure      500  {object}  ErrorMsg
// @Router       /temperature [get]
// @Router       /humidity [get]
func handleQueryByPage(c *gin.Context, collection string, db *mongo.Database) {
	pageUnparsed := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageUnparsed)
	if err != nil {
		lsugar.Error(err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	records, err := GetRecordsByPage(db, collection, int64(page))
	if err != nil {
		lsugar.Error(err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if records == nil {
		// a trick to return empty array
		// https://stackoverflow.com/questions/56200925/return-an-empty-array-instead-of-null-with-golang-for-json-return-with-gin
		var empty = make([]string, 0)
		c.JSON(http.StatusOK, gin.H{"records": empty})
		return
	}
	// RFC3339 is the format of "timestamp"
	c.JSON(http.StatusOK, gin.H{"records": records})
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
// @contact.email  crosstyan@outlook.com

// @license.name  WTFPL
// @license.url   http://www.wtfpl.net/

// @host      localhost:8080
// @BasePath  /
func main() {
	ln, err := net.Listen("tcp", ":1883")
	if err != nil {
		lsugar.Fatal(err.Error())
		return
	}
	docs.SwaggerInfo.Host = "localhost:8080"
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

		r.GET("/temperature",
			func(c *gin.Context) {
				handleQueryByPage(c, "temperature", db)
			})
		r.GET("/humidity", func(c *gin.Context) {
			handleQueryByPage(c, "humidity", db)
		})
		r.POST("/temperature", func(c *gin.Context) {
			handleQuery(c, "temperature", db)
		})
		r.POST("/humidity", func(c *gin.Context) {
			handleQuery(c, "humidity", db)
		})
		// Swagger
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		r.Run(*addr)
	}()

	// gMQTT server
	err = s.Run()
	if err != nil {
		panic(err)
	}
}
