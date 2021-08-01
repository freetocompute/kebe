package server

import (
	"github.com/gin-gonic/gin"
)

func (s *Server) SetupEndpoints(checkForAuthorizedUser gin.HandlerFunc) {
	r := s.engine

	public := r.Group("/dev/api")
	public.POST("/acl/", s.postACL)

	// TODO: document this somehow
	public.GET("/snap-status/:id", s.getStatus)
	public.POST("/acl/verify/", s.verifyACL)

	private := r.Group("/dev/api")
	private.Use(checkForAuthorizedUser)

	private.GET("/account", s.getAccount)
	private.POST("/register-name", s.registerSnapName)

	private.POST("/account/account-key", s.addAccountKey)
	private.POST("/snap-push", s.pushSnap)
	private.POST("/snap-release", s.snapRelease)

	apiV2Private := r.Group("/api/v2")
	apiV2Private.Use(checkForAuthorizedUser)
	apiV2Private.GET("/snaps/:snap/channel-map", s.getSnapChannelMap)

	// TODO: implement /api/v2/snaps/<snap-name>/releases for `snapcraft list-revisions <snap-name>`
}
