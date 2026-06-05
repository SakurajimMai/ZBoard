package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
)

func adminListActiveUsers(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		day, err := activeUsersDateFromQuery(c.Query("date"))
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		params := paginationFromQuery(c)
		rows, total, summary, err := d.Store.ListActiveUsersByDay(c.Request.Context(), day, params)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{
			"date":      day.Format("2006-01-02"),
			"items":     rows,
			"summary":   summary,
			"page":      params.Page,
			"page_size": params.PageSize,
			"total":     total,
		})
	}
}

func activeUsersDateFromQuery(value string) (time.Time, error) {
	today := utcDayStart(time.Now().UTC())
	if value == "" {
		return today, nil
	}
	day, err := time.ParseInLocation("2006-01-02", value, time.UTC)
	if err != nil {
		return time.Time{}, httpx.NewError(http.StatusBadRequest, "bad_request", "日期格式不合法")
	}
	day = utcDayStart(day.UTC())
	oldest := today.AddDate(0, 0, -6)
	if day.Before(oldest) || day.After(today) {
		return time.Time{}, httpx.NewError(http.StatusBadRequest, "bad_request", "只能查看最近 7 天的活跃用户")
	}
	return day, nil
}

func utcDayStart(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
