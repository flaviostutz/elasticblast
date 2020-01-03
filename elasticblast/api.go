package elasticblast

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type HTTPServer struct {
	server *http.Server
	router *mux.Router
}

var apiInvocationsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "elasticblast_api_invocations_total",
	Help: "Total elasticsearch api requests served",
}, []string{
	"entity",
	"status",
})

func NewHTTPServer(blastURL string) *HTTPServer {
	r := mux.NewRouter()
	prometheus.MustRegister(apiInvocationsCounter)

	logrus.Infof("Initializing HTTP Handlers...")
	r.Handle("/metrics", promhttp.Handler())
	initBlast(blastURL)

	s := &http.Server{
		Handler: r,
		Addr:    ":8200",
		// WriteTimeout: 15 * time.Second,
		// ReadTimeout:  15 * time.Second,
	}

	h := &HTTPServer{
		server: s,
		router: r,
	}

	h.setupElasticsearchHandlers()
	return h
}

//Start the main HTTP Server entry
func (s *HTTPServer) Start() error {
	logrus.Infof("Starting HTTP Server (emulated Elasticsearch rest API) on port 8200")
	return s.server.ListenAndServe()
}
