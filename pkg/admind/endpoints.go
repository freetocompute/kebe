package admind

import "github.com/gin-gonic/gin"

func (s *Server) SetupEndpoints(r *gin.Engine) {
	s.engine = r

	r.POST("/v1/admin/account", s.addAccount)
	r.POST("/v1/admin/track", s.addTrack)
}