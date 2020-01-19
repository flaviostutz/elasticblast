package elasticblast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

//METRICS
var invocationHist = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "blast_invocation",
	Help:    "Blast invocations",
	Buckets: []float64{0.1, 1, 10},
}, []string{
	"method",
	"url",
	"status",
})

var blastURL = ""

func initBlast(blastURL0 string) {
	prometheus.MustRegister(invocationHist)
	blastURL = blastURL0
}

func storeDocument(indexname string, mapping string, id string, contents map[string]interface{}) error {
	newdoc := make(map[string]interface{})
	blastID := fmt.Sprintf("%s-%s-%s", indexname, mapping, id)
	newdoc["id"] = blastID
	newdoc["fields"] = contents

	//store as a plain document field for later being searched
	contents["_index"] = indexname
	contents["_mapping"] = mapping
	contents["_id"] = id

	logrus.Debugf("createDocument %v", newdoc)
	wfb, err := json.Marshal(newdoc)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v1/documents", blastURL)
	resp, data, err := sendHTTP(url, wfb, "PUT", "/v1/documents")
	if err != nil {
		logrus.Errorf("Call to Blast POST %s failed. err=%s", url, err)
		return err
	}
	if resp.StatusCode != 200 {
		logrus.Warnf("POST %s call status!=200. resp=%v", url, resp)
		return fmt.Errorf("Failed to create new document. status=%d", resp.StatusCode)
	}
	logrus.Debugf("Document created successfully. data=%v", data)

	return nil
}

//FIXME this code won't work for now because of a Blast bug
//https://github.com/mosuka/blast/issues/123
// func loadDocument(id string) (map[string]interface{}, int, error) {
// 	logrus.Debugf("loadDocument %s", id)

// 	resp, data, err := getHTTP(fmt.Sprintf("%s/v1/documents/%s", blastURL, id), "/v1/documents")
// 	if err != nil {
// 		return nil, resp.StatusCode, fmt.Errorf("GET %s/v1/documents/%s. err=%s", blastURL, id, err)
// 	}
// 	if resp.StatusCode >= 500 {
// 		return nil, resp.StatusCode, fmt.Errorf("Error getting document. id=%s. status=%d", id, resp.StatusCode)
// 	}

// 	var docdata map[string]interface{}
// 	err = json.Unmarshal(data, &docdata)
// 	if err != nil {
// 		logrus.Errorf("Error parsing json. err=%s. body=%s", err, string(data))
// 		return nil, resp.StatusCode, err
// 	}
// 	return docdata, resp.StatusCode, nil
// }

func loadDocument(indexname string, mapping string, id string) (map[string]interface{}, int, error) {
	logrus.Debugf("loadDocument %s", id)

	query := gin.H{
		"search_request": gin.H{
			"from":   0,
			"size":   1,
			"sort":   []string{"-timestamp"},
			"fields": []string{"*"},
			"query": gin.H{
				"query": fmt.Sprintf("+_index:\"%s\" +_mapping:\"%s\" +_id:\"%s\"", indexname, mapping, id),
			},
		},
	}

	return searchDocument(query)
}

func searchDocument(query map[string]interface{}) (map[string]interface{}, int, error) {
	logrus.Debugf("searchDocument %v", query)

	b, err := json.Marshal(query)
	if err != nil {
		return nil, 500, err
	}
	resp, data, err := sendHTTP(fmt.Sprintf("%s/v1/search", blastURL), b, "POST", "/v1/search")
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("POST %s/v1/search. err=%s", blastURL, err)
	}
	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, fmt.Errorf("Couldn't search documents. status=%d", resp.StatusCode)
	}

	var docdata map[string]interface{}
	err = json.Unmarshal(data, &docdata)
	if err != nil {
		logrus.Errorf("Error parsing json. err=%s", err)
		return nil, resp.StatusCode, err
	}
	return docdata, resp.StatusCode, nil
}

func sendHTTP(url string, data []byte, method string, metricsInfo string) (http.Response, []byte, error) {
	startTime := time.Now()
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		logrus.Errorf("HTTP request creation failed. err=%s", err)
		return http.Response{}, []byte{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	logrus.Debugf("%s request=%v", method, req)
	response, err1 := client.Do(req)
	if err1 != nil {
		logrus.Errorf("HTTP request invocation failed. err=%s", err1)
		return http.Response{}, []byte{}, err1
	}

	logrus.Debugf("Response: %v", response)
	datar, _ := ioutil.ReadAll(response.Body)
	logrus.Debugf("Response body: %s", datar)

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		logrus.Debugf("%s status code not ok. status_code=%d", metricsInfo, response.StatusCode)
	}
	invocationHist.WithLabelValues(method, metricsInfo, fmt.Sprintf("%d", response.StatusCode)).Observe(float64(time.Since(startTime).Seconds()))

	return *response, datar, nil
}

func getHTTP(url0 string, metricsInfo string) (http.Response, []byte, error) {
	startTime := time.Now()
	req, err := http.NewRequest("GET", url0, nil)
	if err != nil {
		logrus.Errorf("HTTP request creation failed. err=%s", err)
		return http.Response{}, []byte{}, err
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	logrus.Debugf("GET request=%v", req)
	response, err1 := client.Do(req)
	if err1 != nil {
		logrus.Errorf("HTTP request invocation failed. err=%s", err1)
		invocationHist.WithLabelValues(metricsInfo, "error").Observe(float64(time.Since(startTime).Seconds()))
		return http.Response{}, []byte{}, err1
	}

	// logrus.Debugf("Response: %v", response)
	datar, _ := ioutil.ReadAll(response.Body)
	logrus.Debugf("Response body: %s", datar)

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		logrus.Debugf("%s status code not ok. status_code=%d", metricsInfo, response.StatusCode)
	}
	invocationHist.WithLabelValues("GET", metricsInfo, fmt.Sprintf("%d", response.StatusCode)).Observe(float64(time.Since(startTime).Seconds()))

	return *response, datar, nil
}
