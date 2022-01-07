package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	l "github.com/crosstyan/mqtt-to-ws/logger"
	"github.com/crosstyan/mqtt-to-ws/model"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

var logger = l.Lsugar

// Chain33Info
// Optional
type Chain33Info struct {
	PrivKey string `json:"priv_key" example:"cc38546e9e659d15e6b4893f0ab32a06d103931a8230b0bde71459d2b27d6944"`
	Url     string `json:"url" example:"http://127.0.0.1:8801"`
}

type DateRangeRequest struct {
	// Page is from 1 to infinity
	Page *int64 `json:"page,omitempty" example:"1"`
	// Time RFC3339
	Start *string `json:"start" example:"2020-01-01T00:00:00Z" validate:"required"`
	// Time RFC3339
	End       *string      `json:"end,omitempty" example:"2022-01-01T00:00:00Z"`
	Info      *Chain33Info `json:"chain,omitempty"`
	IsDescend *bool        `json:"descend,omitempty" example:"true"`
}

type ErrorMsg struct {
	// Error message
	Err string `json:"error" example:"error message"`
}

type ResponseMsg struct {
	Records []model.MQTTRecord `json:"records"`
}

// HandleQuery
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
func HandleQuery(c *gin.Context, collection string, db *mongo.Database) {
	var dateRequest DateRangeRequest
	var records []model.MQTTRecord
	err := c.BindJSON(&dateRequest)
	if err != nil {
		logger.Error(err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var page int64
	if dateRequest.Page == nil || *dateRequest.Page <= 0 {
		page = 1
	} else {
		page = *dateRequest.Page
	}
	var tEnd time.Time
	var tStart time.Time
	// default descending
	var isDescend bool = true
	if dateRequest.IsDescend != nil {
		isDescend = *dateRequest.IsDescend
	}
	tStart, err = time.Parse(time.RFC3339, *dateRequest.Start)
	if err != nil {
		logger.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: Refactor this part
	if dateRequest.End != nil {
		tEnd, err = time.Parse(time.RFC3339, *dateRequest.End)
		if err != nil {
			logger.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		records, err = model.GetRecordsBetween(db, collection, tStart, tEnd, page, isDescend)
		if err != nil {
			logger.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		records, err = model.GetRecordsFrom(db, collection, tStart, page, isDescend)
		if err != nil {
			logger.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if dateRequest.Info != nil && records != nil {
		content, err := json.Marshal(records)
		if err != nil {
			logger.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		err = SaveToBlockchain(content, dateRequest.Info.PrivKey, dateRequest.Info.Url)
		if err != nil {
			logger.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
	}
	if records == nil {
		var empty = make([]string, 0)
		c.JSON(http.StatusOK, gin.H{"records": empty})
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": records})
}

// HandleQueryByPage
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
func HandleQueryByPage(c *gin.Context, collection string, db *mongo.Database) {
	pageUnparsed := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageUnparsed)
	if err != nil {
		logger.Error(err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	records, err := model.GetRecordsByPage(db, collection, int64(page))
	if err != nil {
		logger.Error(err.Error())
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
