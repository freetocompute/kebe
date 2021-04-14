package store

import "github.com/gin-gonic/gin"

func (s *Store) SetupEndpoints(r *gin.Engine) {
	r.GET("/api/v1/snaps/sections", s.getSnapSections)
	r.GET("/api/v1/snaps/names", s.getSnapNames)
	r.GET("/v2/snaps/find", s.findSnap)
	r.POST("/v2/snaps/refresh", s.snapRefresh)
	r.GET("/download/snaps/:filename", s.snapDownload)
	r.GET("/api/v1/snaps/assertions/snap-revision/:sha3384digest", s.getSnapRevisionAssertion)
	r.GET("/api/v1/snaps/assertions/snap-declaration/16/:snap-id", s.getSnapDeclarationAssertion)
	r.GET("/api/v1/snaps/assertions/account-key/:key", s.getAccountKey)
	r.GET("/api/v1/snaps/assertions/account/:id", s.getAccountAssertion)
}
