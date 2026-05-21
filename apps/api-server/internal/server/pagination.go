package server

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/store"
)

func paginationFromQuery(c *gin.Context) store.PageParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return store.NormalizePage(store.PageParams{Page: page, PageSize: pageSize})
}
