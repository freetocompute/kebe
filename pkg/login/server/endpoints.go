package server

import "github.com/gin-gonic/gin"

func (s *Server) SetupEndpoints(r *gin.Engine) {
	r.POST("/api/v2/tokens/discharge", s.dischargeTokens)
}