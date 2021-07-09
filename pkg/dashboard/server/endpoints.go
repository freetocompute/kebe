package server

import (
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func (s *Server) SetupEndpoints(r *gin.Engine) {
	rootKey := config.MustGetString(configkey.MacaroonRootKey)

	public := r.Group("/dev/api")
	public.POST("/acl/", s.postACL)
	// TODO: document this somehow
	public.GET("/snap-status/:id", s.getStatus)

	private := r.Group("/dev/api")
	private.Use(middleware.CheckForAuthorizedUserWithMacaroons(s.db, rootKey))
	private.GET("/account", s.getAccount)
	private.POST("/register-name", s.registerSnapName)
	private.POST("/account/account-key", s.addAccountKey)
	private.POST("/snap-push", s.pushSnap)

	private.POST("/snap-release", s.snapRelease)

	apiV2Private := r.Group("/api/v2")
	apiV2Private.Use(middleware.CheckForAuthorizedUserWithMacaroons(s.db, rootKey))
	apiV2Private.GET("/snaps/:snap/channel-map", s.getSnapChannelMap)

	// TODO: implement /api/v2/snaps/<snap-name>/releases for `snapcraft list-revisions <snap-name>`
	// TODO: implement /api/v2/snaps/<snap-name>/channel-map for `snapcraft list-tracks <snap-name>`
	// TODO: implement /dev/api/snap-release/ for `snapcraft release <snap-name> <revision> <channels>`
	// TODO: implement /dev/api/snap-push/ for `snapcraft upload <snap-name>`

	// TODO: implement "/api/v1/snaps/auth/request-id"
	// TODO: implement "/api/v1/snaps/auth/devices"
}