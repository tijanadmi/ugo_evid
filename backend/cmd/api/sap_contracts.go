package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
)

type listSapUgovoriPagedRequest struct {
	Dobavljac string `form:"dobavljac"`
	PageID    int32  `form:"page_id" binding:"required,min=1"`
	PageSize  int32  `form:"page_size" binding:"required,min=5,max=100"`
}

type listSapUgovoriPagedResponse struct {
	Total   int                 `json:"total"`
	Ugovori []*models.SapUgovor `json:"ugovori"`
}

func (server *Server) GetSapUgovoriPaged(ctx *gin.Context) {
	var req listSapUgovoriPagedRequest

	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// izračunaj offset
	limit := int(req.PageSize)
	offset := int((req.PageID - 1) * req.PageSize)

	// poziv repo funkcije
	ugovori, count, err := server.store.GetSapUgovoriPaged(ctx, offset, limit, req.Dobavljac)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// response
	rsp := &listSapUgovoriPagedResponse{
		Total: count,
		// Ugovori: []*models.SapUgovor{},
		Ugovori: append([]*models.SapUgovor{}, ugovori...),
	}

	/*for _, u := range ugovori {
	    rsp.Ugovori = append(rsp.Ugovori,u)
	}*/

	ctx.JSON(http.StatusOK, rsp)

}
