package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/admind"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Server struct {
	engine       *gin.Engine
	oauth2Config *oauth2.Config
	userInfo     *admind.UserInfo
	done         chan struct{}
	port         int32
}

var login = &cobra.Command{
	Use:   "login",
	Short: "login",
	Run: func(cmd *cobra.Command, args []string) {
		s := &Server{
			done: make(chan struct{}),
		}
		port := config.MustGetInt32(configkey.AdminCLILoginPort)
		s.Init(port)
		openbrowser("http://localhost:" + strconv.Itoa(int(port)) + "/")
		go func() {
			s.Run()
		}()

		<-s.done
		s.Stop()
	},
}

func (s *Server) Init(port int32) {
	s.port = port
	logrus.SetLevel(logrus.TraceLevel)
	config.LoadConfig()

	logLevelConfig := viper.GetString(configkey.LogLevel)
	l, errLevel := logrus.ParseLevel(logLevelConfig)
	if errLevel != nil {
		logrus.Error(errLevel)
	} else {
		logrus.SetLevel(l)
	}

	// Setup gin and routes
	r := gin.Default()
	if viper.GetBool(configkey.DebugMode) {
		logrus.Info("Debug mode enabled")
		r.Use(middleware.RequestLoggerMiddleware())
	} else {
		logrus.Info("Debug mode disabled")
	}

	s.engine = r
	r.Use(static.Serve("/", static.LocalFile("./static/starfruit", false)))
	r.GET("/callback/login", s.loginCallback)
	r.GET("/login", s.login)
	r.GET("/api/account", s.account)

	s.oauth2Config = &oauth2.Config{
		ClientID:     config.MustGetString(configkey.OIDCClientId),
		ClientSecret: config.MustGetString(configkey.OIDCClientSecret),
		Endpoint: oauth2.Endpoint{
			TokenURL: config.MustGetString(configkey.OIDCProviderURL) + "/protocol/openid-connect/token",
			AuthURL:  config.MustGetString(configkey.OIDCProviderURL) + "/protocol/openid-connect/auth",
		},
		RedirectURL: "http://localhost:" + strconv.Itoa(int(port)) + "/callback/login",
		Scopes:      []string{"openid", "email", "profile"},
	}
}

func (s *Server) Run() {
	adminLoginPort := viper.GetInt(configkey.AdminCLILoginPort)
	_ = s.engine.Run(fmt.Sprintf(":%d", adminLoginPort))
}

func (s *Server) Stop() {
}

func (s *Server) loginCallback(c *gin.Context) {
	fmt.Printf("%+v", c.Request)

	authCode := c.Query("code")
	token, err := s.oauth2Config.Exchange(context.Background(), authCode)
	if err != nil {
		log.Fatal(err)
	}

	client := s.oauth2Config.Client(context.Background(), token)

	response, err := client.Get(config.MustGetString(configkey.OIDCProviderURL) + "/protocol/openid-connect/userinfo")
	if err != nil {
		panic(err)
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	var userInfo admind.UserInfo
	_ = json.Unmarshal(bytes, &userInfo)

	s.userInfo = &userInfo

	loginInfo := admind.LoginInfo{
		UserInfo: userInfo,
		Token:    *token,
	}

	loginInfoBytes, _ := json.Marshal(&loginInfo)
	err2 := ioutil.WriteFile(LoginConfigFilename, loginInfoBytes, 0600)
	if err2 != nil {
		panic(err)
	}

	// TODO: use a cookie to store a session ID and we can use that to look up the user

	c.Redirect(http.StatusTemporaryRedirect, "http://localhost:"+strconv.Itoa(int(s.port)))

	go func() {
		time.Sleep(3 * time.Second)
		s.done <- struct{}{}
	}()
}

func (s *Server) login(c *gin.Context) {
	url := s.oauth2Config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (s *Server) account(c *gin.Context) {
	if s.userInfo != nil {
		c.JSON(200, s.userInfo)
		return
	} else {
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

// Source: https://gist.github.com/hyg/9c4afcd91fe24316cbf0
func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}
