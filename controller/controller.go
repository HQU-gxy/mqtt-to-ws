package controller

import (
	"net/http"
	"strconv"
	"time"

	l "github.com/crosstyan/mqtt-to-ws/logger"
	"github.com/crosstyan/mqtt-to-ws/model"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

var logger = l.Lsugar

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
	Records []model.MQTTRecord `json:"records"`
}

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
	var dateRange DateRangeRequest
	err := c.BindJSON(&dateRange)
	if err != nil {
		logger.Error(err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var page int64
	if dateRange.Page == nil || *dateRange.Page <= 0 {
		page = 1
	} else {
		page = *dateRange.Page
	}
	var tEnd time.Time
	var tStart time.Time
	tStart, err = time.Parse(time.RFC3339, *dateRange.Start)
	if err != nil {
		logger.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: Refactor this part
	if dateRange.End != nil {
		tEnd, err = time.Parse(time.RFC3339, *dateRange.End)
		if err != nil {
			logger.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		records, err := model.GetRecordsBetween(db, collection, tStart, tEnd, page)
		if err != nil {
			logger.Error(err)
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
		records, err := model.GetRecordsFrom(db, collection, tStart, page)
		if err != nil {
			logger.Error(err)
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
