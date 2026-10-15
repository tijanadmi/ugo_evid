package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
	"strconv"
)

func (s *Server) Calendar(c *gin.Context) {
	if _, _, ok := s.registrationIdentity(c); !ok {
		return
	}
	year, e := strconv.Atoi(c.Param("year"))
	if e != nil || year < 1900 || year > 2200 {
		c.JSON(400, gin.H{"error": "Godina mora biti izmedju 1900 i 2200."})
		return
	}
	if c.Request.Method == "GET" {
		data, e := s.store.GetCalendar(c.Request.Context(), year)
		if e != nil {
			maintenanceError(c, e)
			return
		}
		c.JSON(200, data)
		return
	}
	var m models.CalendarChange
	if e = c.ShouldBindJSON(&m); e != nil {
		c.JSON(400, gin.H{"error": "Neispravni podaci kalendara."})
		return
	}
	if e = s.store.ChangeCalendar(c.Request.Context(), year, m); e != nil {
		maintenanceError(c, e)
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}
