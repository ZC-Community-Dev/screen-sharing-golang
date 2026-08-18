package httpapi

import "github.com/gin-gonic/gin"

const (
	CodeLinkNotFound          = "link_not_found"
	CodeLinkInvalid           = "link_invalid"
	CodePresenterUnauthorized = "presenter_unauthorized"
	CodePresenterConflict     = "presenter_conflict"
	CodeShareConflict         = "share_conflict"
	CodeMediaNotReady         = "media_not_ready"
	CodeMediaConflict         = "media_conflict"
	CodeMediaCapacity         = "media_capacity"
	CodeInvalidSDP            = "invalid_sdp"
	CodeMediaSessionNotFound  = "media_session_not_found"
	CodeTransportInvalid      = "transport_invalid"
	CodeTransportNotAllowed   = "transport_not_allowed"
	CodeTransportMismatch     = "publication_transport_mismatch"
	CodeMediaCodecUnsupported = "media_codec_unsupported"
	CodeMediaStartTimeout     = "media_start_timeout"
	CodeMediaProtocolError    = "media_protocol_error"
	CodeMediaSlowConsumer     = "media_slow_consumer"
	CodeWebSocketUpgrade      = "websocket_upgrade_required"
	CodeInternalError         = "internal_error"
	CodeRouteNotFound         = "route_not_found"
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
