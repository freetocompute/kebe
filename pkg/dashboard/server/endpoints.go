package server

import (
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func (s *Server) SetupEndpoints(r *gin.Engine) {
	public := r.Group("/dev/api")
	public.POST("/acl/", s.postACL)

	rootKey := config.MustGetString(configkey.MacaroonRootKey)

	private := r.Group("/dev/api")
	private.Use(middleware.CheckForAuthorizedUserWithMacaroons(s.db, rootKey))
	private.GET("/account", s.getAccount)
	private.POST("/register-name", s.registerSnapName)
}