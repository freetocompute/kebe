package middleware

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
)

func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.ContentType() == "multipart/form-data" {
			logrus.Infof("Skipping body for multipart/form-data")
		} else {
			var buf bytes.Buffer
			tee := io.TeeReader(c.Request.Body, &buf)
			body, _ := ioutil.ReadAll(tee)
			c.Request.Body = ioutil.NopCloser(&buf)
			fmt.Println("Body:")

			fmt.Println(string(body))
		}

		fmt.Println("Header:")
		fmt.Println(c.Request.Header)
		c.Next()
	}
}
