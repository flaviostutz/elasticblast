package elasticblast

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func (h *HTTPServer) setupElasticsearchHandlers() {
	h.router.HandleFunc("/_cluster/health", getClusterHealth).Methods("GET")

	// h.router.HandleFunc("/_template/{template}", headTemplate).Methods("HEAD")
	// h.router.PUT("/_template/:template", putTemplate)
	// h.router.HEAD("/:indexname/_mapping/:mapping", headIndexMapping)
	// h.router.PUT("/:indexname/_mapping/:mapping", putIndexMapping)
	h.router.HandleFunc("/{indexname}/{mapping}/{id}", headDocument).Methods("HEAD")
	h.router.HandleFunc("/{indexname}/{mapping}/{id}", putDocument).Methods("PUT")
	// h.router.HEAD("/:indexname", headIndex)
	// h.router.PUT("/:indexname", putIndex)
}

func getClusterHealth(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("getClusterHealth")
	st, exists := r.URL.Query()["wait_for_status"]
	status := ""
	if exists {
		status = st[0]
	}
	jsonWrite(w, http.StatusOK, gin.H{"cluster_name": "docker-cluster", "status": status, "timed_out": false, "number_of_nodes": 1, "number_of_data_nodes": 1, "active_primary_shards": 0, "active_shards": 0, "relocating_shards": 0, "initializing_shards": 0, "unassigned_shards": 0, "delayed_unassigned_shards": 0, "number_of_pending_tasks": 0, "number_of_in_flight_fetch": 0, "task_max_waiting_in_queue_millis": 0, "active_shards_percent_as_number": 100.0})
}

func headDocument(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	mapping := param(r, "mapping", "")
	id := param(r, "id", "")
	docid := fmt.Sprintf("_doc-%s-%s-%s", indexname, mapping, id)
	logrus.Debugf("headDocument docid=%s", docid)
	_, status, err := loadDocument(docid)
	if err != nil {
		// c.JSON(http.StatusOK, gin.H{})
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	jsonWrite(w, status, gin.H{})
}

func putDocument(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	mapping := param(r, "mapping", "")
	id := param(r, "id", "")
	docid := fmt.Sprintf("_doc-%s-%s-%s", indexname, mapping, id)
	logrus.Debugf("putDocument docid=%s", docid)

	bb, _ := ioutil.ReadAll(r.Body)

	var docdata map[string]interface{}
	err := json.Unmarshal(bb, &docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	docdata["id"] = docid
	setType(docdata, fmt.Sprintf("%s-%s", indexname, mapping))
	err = storeDocument(docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	jsonWrite(w, http.StatusCreated, gin.H{})
}

// func headIndex(c *gin.Context) {
// 	indexname := c.Param("indexname")
// 	s := fmt.Sprintf("_index-%s", indexname)
// 	logrus.Debugf("headIndex index=%s", s)
// 	_, status, err := loadDocument(s)
// 	if err != nil {
// 		// c.JSON(http.StatusOK, gin.H{})
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
// 		return
// 	}
// 	c.JSON(status, gin.H{})
// }

// func putIndex(c *gin.Context) {
// 	indexname := c.Param("indexname")
// 	mapping := c.Param("mapping")
// 	docid := fmt.Sprintf("_index-%s-%s", indexname, mapping)
// 	logrus.Debugf("putIndex index=%s", docid)

// 	bb, _ := ioutil.ReadAll(c.Request.Body)

// 	var docdata map[string]interface{}
// 	err := json.Unmarshal(bb, &docdata)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
// 		return
// 	}

// 	docdata["id"] = docid
// 	setType(docdata, "_esindex")
// 	err = storeDocument(docdata)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
// 		return
// 	}
// 	c.JSON(http.StatusCreated, gin.H{})
// }

// func headIndexMapping(c *gin.Context) {
// 	indexname := c.Param("indexname")
// 	mapping := c.Param("mapping")
// 	docid := fmt.Sprintf("_index-%s-%s", indexname, mapping)
// 	logrus.Debugf("headIndexMapping index-mapping=%s", docid)

// 	_, status, err := loadDocument(docid)
// 	if err != nil {
// 		// c.JSON(http.StatusOK, gin.H{})
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
// 		return
// 	}
// 	c.JSON(status, gin.H{})
// }

// func putIndexMapping(c *gin.Context) {
// 	indexname := c.Param("indexname")
// 	mapping := c.Param("mapping")
// 	docid := fmt.Sprintf("_index-%s-%s", indexname, mapping)
// 	logrus.Debugf("putIndexMapping index-mapping=%s", docid)

// 	bb, _ := ioutil.ReadAll(c.Request.Body)

// 	var docdata map[string]interface{}
// 	err := json.Unmarshal(bb, &docdata)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
// 		return
// 	}

// 	docdata["id"] = docid
// 	setType(docdata, "_esindexmapping")
// 	err = storeDocument(docdata)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
// 		return
// 	}

// 	c.JSON(http.StatusCreated, gin.H{})
// }

// func headTemplate(c *gin.Context) {
// 	t := c.Param("template")
// 	s := fmt.Sprintf("_template-%s", t)
// 	logrus.Debugf("headTemplate template=%s", s)

// 	_, status, err := loadDocument(s)
// 	if err != nil {
// 		//FIXME fix Blast and remove this later
// 		c.JSON(http.StatusOK, gin.H{})
// 		// c.JSON(http.StatusInternalServerError, gin.H{"error": err})
// 		return
// 	}
// 	c.JSON(status, gin.H{})
// }

// func putTemplate(c *gin.Context) {
// 	t := c.Param("template")
// 	logrus.Debugf("putTemplate template=%s", t)

// 	bb, _ := ioutil.ReadAll(c.Request.Body)

// 	var docdata map[string]interface{}
// 	err := json.Unmarshal(bb, &docdata)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
// 		return
// 	}

// 	docid := fmt.Sprintf("_template-%s", t)
// 	docdata["id"] = docid
// 	setType(docdata, "_esindextemplate")
// 	err = storeDocument(docdata)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
// 		return
// 	}

// 	c.JSON(http.StatusCreated, gin.H{})
// }

func setType(docdata map[string]interface{}, dtype string) {
	fields, exists := docdata["fields"]
	if !exists {
		fields = make(map[string]interface{})
		docdata["fields"] = fields
	}
	f := fields.(map[string]interface{})
	f["_type"] = dtype
}

func jsonWrite(w http.ResponseWriter, statusCode int, result interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(result)
}

func param(r *http.Request, paramName string, defaultValue string) string {
	params := mux.Vars(r)
	value, exists := params[paramName]
	if !exists {
		return defaultValue
	}
	return value
}
