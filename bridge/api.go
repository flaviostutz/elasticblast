package backtor

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	cors "github.com/itsjamie/gin-cors"
)

type HTTPServer struct {
	server *http.Server
	router *gin.Engine
}

var apiInvocationsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "backtor_api_invocations_total",
	Help: "Total api requests served",
}, []string{
	"entity",
	"status",
})

func NewHTTPServer() *HTTPServer {
	router := gin.Default()

	router.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET",
		RequestHeaders:  "Origin, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          24 * 3600 * time.Second,
		Credentials:     false,
		ValidateHeaders: false,
	}))

	h := &HTTPServer{server: &http.Server{
		Addr:    ":6000",
		Handler: router,
	}, router: router}

	prometheus.MustRegister(apiInvocationsCounter)

	logrus.Infof("Initializing HTTP Handlers...")
	h.setupMaterializedHandlers()
	h.setupBackupSpecHandlers()
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return h
}

//Start the main HTTP Server entry
func (s *HTTPServer) Start() error {
	logrus.Infof("Starting HTTP Server on port 6000")
	return s.server.ListenAndServe()
}
