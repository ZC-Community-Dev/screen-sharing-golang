package httpapi

import "github.com/gin-gonic/gin"

const (
	CodeLinkNotFound          = "link_not_found"
	CodeLinkInvalid           = "link_invalid"
	CodePresenterUnauthorized = "presenter_unauthorized"
	CodePresenterConflict     = "presenter_conflict"
	CodeShareConflict         = "share_conflict"
	CodeInternalError         = "internal_error"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(c *gin.Context, status int, code, message string) {
	var body errorBody
	body.Error.Code = code
	body.Error.Message = message
	c.JSON(status, body)
}
