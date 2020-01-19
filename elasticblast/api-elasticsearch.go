package elasticblast

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func (h *HTTPServer) setupElasticsearchHandlers() {
	h.router.HandleFunc("/_cluster/health", getClusterHealth).Methods("GET")

	h.router.HandleFunc("/{indexname}/_mapping/{mapping}", headIndexMapping).Methods("HEAD")
	h.router.HandleFunc("/{indexname}/_mapping/{mapping}", putIndexMapping).Methods("PUT")

	h.router.HandleFunc("/{indexname}/{mapping}/{id}", headDocument).Methods("HEAD")
	h.router.HandleFunc("/{indexname}/{mapping}/{id}", getDocument).Methods("GET")
	h.router.HandleFunc("/{indexname}/{mapping}/{id}", putDocument).Methods("PUT")
	h.router.HandleFunc("/{indexname}/{mapping}/{id}/_update", updateDocument).Methods("POST")

	h.router.HandleFunc("/{indexname}/{mapping}/_search", postSearch).Methods("POST")

	h.router.HandleFunc("/_template/{template}", headTemplate).Methods("HEAD")
	h.router.HandleFunc("/_template/{template}", putTemplate).Methods("PUT")

	h.router.HandleFunc("/{indexname}", headIndex).Methods("HEAD")
	h.router.HandleFunc("/{indexname}", putIndex).Methods("PUT")
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

	logrus.Debugf("headDocument indexname=%s mapping=%s id=%s", indexname, mapping, id)
	_, status, err := loadDocument(indexname, mapping, id)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	jsonWrite(w, status, gin.H{})
}

func getDocument(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	mapping := param(r, "mapping", "")
	id := param(r, "id", "")

	logrus.Debugf("getDocument indexname=%s mapping=%s id=%s", indexname, mapping, id)
	_, status, err := loadDocument(indexname, mapping, id)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	//transform Blast query result in document search result

	jsonWrite(w, status, gin.H{})
}

func putDocument(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	mapping := param(r, "mapping", "")
	id := param(r, "id", "")

	logrus.Debugf("putDocument indexname=%s mapping=%s id=%s", indexname, mapping, id)

	bb, _ := ioutil.ReadAll(r.Body)

	var docdata map[string]interface{}
	err := json.Unmarshal(bb, &docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	err = storeDocument(indexname, mapping, id, docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	jsonWrite(w, http.StatusCreated, gin.H{"_index": indexname, "_type": mapping, "_id": id, "_version": 1, "result": "created", "_shards": gin.H{"total": 2, "successful": 1, "failed": 0}, "created": true})
}

func updateDocument(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	mapping := param(r, "mapping", "")
	id := param(r, "id", "")

	logrus.Debugf("updateDocument indexname=%s mapping=%s id=%s", indexname, mapping, id)

	bb, _ := ioutil.ReadAll(r.Body)

	var docdata map[string]interface{}
	err := json.Unmarshal(bb, &docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	doc, exists := docdata["doc"]
	if !exists {
		jsonWrite(w, http.StatusBadRequest, gin.H{"error": "doc field doesn't exist in json from post"})
		return
	}

	doc0 := doc.(map[string]interface{})

	archived, exists := doc0["archived"]
	isarchived := false
	if exists {
		isarchived = archived.(bool)
	}

	rawJSON, exists := doc0["rawJSON"]
	if !exists {
		jsonWrite(w, http.StatusBadRequest, gin.H{"error": "rawJSON field doesn't exist in json from post"})
		return
	}
	rawJSON0 := rawJSON.(string)
	rawJSONUnesc := strings.ReplaceAll(rawJSON0, "\\\"", "\"")

	var newdocdata map[string]interface{}
	err = json.Unmarshal([]byte(rawJSONUnesc), &newdocdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	newdocdata["_archived"] = isarchived

	err = storeDocument(indexname, mapping, id, newdocdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	jsonWrite(w, http.StatusCreated, gin.H{"_index": indexname, "_type": mapping, "_id": id, "_version": 1, "result": "updated", "_shards": gin.H{"total": 2, "successful": 1, "failed": 0}})
}

func headTemplate(w http.ResponseWriter, r *http.Request) {
	template := param(r, "template", "")
	s := fmt.Sprintf("_template-%s", template)
	logrus.Debugf("headTemplate template=%s", s)

	_, status, err := loadDocument("_internal", "_esindextemplate", s)
	if err != nil {
		//FIXME fix Blast and remove this later
		// jsonWrite(w, http.StatusOK, gin.H{})
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	jsonWrite(w, status, gin.H{})
}

func putTemplate(w http.ResponseWriter, r *http.Request) {
	template := param(r, "template", "")
	logrus.Debugf("putTemplate template=%s", template)

	bb, _ := ioutil.ReadAll(r.Body)

	var docdata map[string]interface{}
	err := json.Unmarshal(bb, &docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	err = storeDocument("_internal", "_esindextemplate", template, docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	jsonWrite(w, http.StatusCreated, gin.H{"acknowledged": true})
}

func headIndexMapping(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	mapping := param(r, "mapping", "")
	id := fmt.Sprintf("%s-%s", indexname, mapping)
	logrus.Debugf("headIndexMapping id=%s", id)

	_, status, err := loadDocument("_internal", "_esindexmapping", id)
	if err != nil {
		// c.JSON(http.StatusOK, gin.H{})
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	jsonWrite(w, status, gin.H{})
}

func putIndexMapping(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	mapping := param(r, "mapping", "")
	id := fmt.Sprintf("%s-%s", indexname, mapping)
	logrus.Debugf("putIndexMapping id=%s", id)

	bb, _ := ioutil.ReadAll(r.Body)

	var docdata map[string]interface{}
	err := json.Unmarshal(bb, &docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	err = storeDocument("_internal", "_esindexmapping", id, docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	jsonWrite(w, http.StatusCreated, gin.H{"acknowledged": true})
}

func headIndex(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	id := fmt.Sprintf("%s", indexname)
	logrus.Debugf("headIndex id=%s", id)
	_, status, err := loadDocument("_internal", "_esindex", id)
	if err != nil {
		// c.JSON(http.StatusOK, gin.H{})
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	jsonWrite(w, status, gin.H{})
}

func putIndex(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	id := fmt.Sprintf("%s", indexname)
	logrus.Debugf("putIndex index=%s", indexname)

	docdata := make(map[string]interface{})

	err := storeDocument("_internal", "_esindex", id, docdata)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	jsonWrite(w, http.StatusCreated, gin.H{"acknowledged": true, "shards_acknowledged": true, "index": indexname})
}

func postSearch(w http.ResponseWriter, r *http.Request) {
	indexname := param(r, "indexname", "")
	mapping := param(r, "mapping", "")
	logrus.Debugf("postSearch index=%s mapping=%s", indexname, mapping)

	bb, _ := ioutil.ReadAll(r.Body)

	var query map[string]interface{}
	err := json.Unmarshal(bb, &query)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	logrus.Debugf("ES QUERY: %v", query)

	//PARSE ES QUERY AND CREATE A BLAST QUERY (this is not a compiler. we are being simplist here for some known cases of queries)
	// value, dt, os, err := jsonparser.Get(bb, "query", "bool", "must")
	fieldsBlast := []string{}
	blastQuery := ""
	_, err1 := jsonparser.ArrayEach(bb, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		logrus.Debugf("Value: '%s'\n Type: %s\n", string(value), dataType)

		//process 'query_string' field
		q, _, _, err2 := jsonparser.Get(value, "query_string", "query")
		if err2 == nil {
			qs := string(q)

			//use only first term
			qs1 := strings.Split(qs, " AND ")
			for _, qv := range qs1 {
				//specific field query
				if strings.Contains(qv, ":") {
					av := strings.Split(qv, ":")
					//looks like a timestamp range query and we don't support yet. ignore term
					if !strings.Contains(av[1], "[") {
						blastQuery = fmt.Sprintf("%s +%s:%s", blastQuery, strings.Trim(av[0], " "), av[1])
					}

				} else {
					//all fields query
					blastQuery = fmt.Sprintf("%s +_all:%s", blastQuery, qv)
				}
			}

			//process 'query_string' selected fields
			_, err6 := jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				fieldName := string(value)
				fieldsBlast = append(fieldsBlast, fieldName)
			}, "query_string", "fields")
			if err6 != nil {
				logrus.Debugf("No fields found for query string. err6=%s", err6)
			}
			if len(fieldsBlast) == 0 {
				fieldsBlast = []string{"*"}
			}

			return
		}

		//process field query matches
		_, err3 := jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			blastQuery = processTerms(value, blastQuery)
		}, "bool", "must", "[0]", "bool", "must")

		_, err4 := jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			blastQuery = processTerms(value, blastQuery)
		}, "bool", "must")

		if err3 != nil && err4 != nil {
			logrus.Warnf("No valid field terms found. err3=%s err4=%s", err3, err4)
			return
		}
	}, "query", "bool", "must")

	if err1 != nil {
		logrus.Debugf("Json query parse error. Ignoring query terms. err=%s", err1)
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	logrus.Debugf("BLAST QUERY=%s", blastQuery)

	//SORT BLAST
	sortBlast := []string{}
	_, err5 := jsonparser.ArrayEach(bb, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		jsonparser.ObjectEach(value, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			fieldName := string(key)
			fieldOrder := "+"
			order, _, _, err9 := jsonparser.Get(value, "order")
			if err9 == nil {
				if string(order) == "desc" {
					fieldOrder = "-"
				}
			}
			sortBlast = append(sortBlast, fmt.Sprintf("%s%s", fieldOrder, fieldName))
			return nil
		})
	}, "sort")
	if err5 != nil {
		logrus.Debugf("Not sort attribute found. err=%s", err)
	}

	// logrus.Debugf("SEARCH value=%s dt=%s os=%v err=%s", value, dt, os, err)

	//ES REQUEST QUERY SAMPLE FROM CONDUCTOR
	//POST /conductor/workflow/_search?typed_keys=true&ignore_unavailable=false&expand_wildcards=open&allow_no_indices=true&search_type=query_then_fetch&batched_reduce_size=512"
	//req_body:"{"from":0,"size":100,"query":{"bool":{"must":[{"query_string":{"query":"*","fields":[],"use_dis_max":true,"tie_breaker":0.0,"default_operator":"or","auto_generate_phrase_queries":false,"max_determinized_states":10000,"enable_position_increments":true,"fuzziness":"AUTO","fuzzy_prefix_length":0,"fuzzy_max_expansions":50,"phrase_slop":0,"escape":false,"split_on_whitespace":true,"boost":1.0}},{"bool":{"must":[{"match_all":{"boost":1.0}}],"disable_coord":false,"adjust_pure_negative":true,"boost":1.0}}],"disable_coord":false,"adjust_pure_negative":true,"boost":1.0}},"sort":[{"startTime":{"order":"desc"}}]}"

	//Transform ES query to Blast query
	queryBlast := gin.H{
		"search_request": gin.H{
			"from":   0,
			"size":   100,
			"sort":   sortBlast,
			"fields": fieldsBlast,
			"query": gin.H{
				"query": fmt.Sprintf("+_index:\"%s\" +_mapping:\"%s\" %s", indexname, mapping, blastQuery),
			},
		},
	}

	//BLAST QUERY
	//{ "search_request": { "query": { "query": "+_all:internet" }, "size": 10, "from": 0, "fields": [ "*" ], "sort": [ "-_score", "_id", "-timestamp" ], "facets": { "Type count": { "size": 10, "field": "_type" }, "Timestamp range": { "size": 10, "field": "timestamp", "date_ranges": [ { "name": "2001 - 2010", "start": "2001-01-01T00:00:00Z", "end": "2010-12-31T23:59:59Z" }, { "name": "2011 - 2020", "start": "2011-01-01T00:00:00Z", "end": "2020-12-31T23:59:59Z" } ] } }, "highlight": { "style": "html", "fields": [ "title", "text" ] } }}

	searchResult, status, err1 := searchDocument(queryBlast)
	if err1 != nil {
		jsonWrite(w, status, gin.H{"error": err1})
		return
	}

	//BLAST RESPONSE
	//{ "search_result": { "status": { "total": 1, "failed": 0, "successful": 1 }, "request": { "query": { "query": "+_all:internet" }, "size": 10, "from": 0, "highlight": { "style": "html", "fields": [ "title", "text" ] }, "fields": [ "*" ], "facets": { "Timestamp range": { "size": 10, "field": "timestamp", "date_ranges": [ { "end": "2010-12-31T23:59:59Z", "name": "2001 - 2010", "start": "2001-01-01T00:00:00Z" }, { "end": "2020-12-31T23:59:59Z", "name": "2011 - 2020", "start": "2011-01-01T00:00:00Z" } ] }, "Type count": { "size": 10, "field": "_type" } }, "explain": false, "sort": [ "-_score", "_id", "-timestamp" ], "includeLocations": false, "search_after": null, "search_before": null }, "hits": [], "total_hits": 0, "max_score": 0, "took": 229700, "facets": { "Timestamp range": { "field": "timestamp", "total": 0, "missing": 0, "other": 0 }, "Type count": { "field": "_type", "total": 0, "missing": 0, "other": 0 } } }}
	logrus.Debugf("BLAST RESPONSE: %v", searchResult)

	sr, ok := searchResult["search_result"]
	if !ok {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Blast didn't not return a search_result field. response=%v", searchResult)})
		return
	}
	sr0 := sr.(map[string]interface{})
	hits, ok := sr0["hits"]
	if !ok {
		jsonWrite(w, http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Blast didn't not return a hits field. response=%v", searchResult)})
		return
	}
	took, ok := sr0["took"]
	tookES := 123
	if !ok {
		logrus.Debugf("Not 'took' field found from Blast response")
	}
	tookES = int((took.(float64)) / float64(1000000))

	//convert BLAST response to ES response
	hitsES := make([]gin.H, 0)
	hits0 := hits.([]interface{})
	for _, v := range hits0 {
		v0 := v.(map[string]interface{})
		vt, ok := v0["fields"]
		if !ok {
			jsonWrite(w, http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Blast didn't not return a result item with 'fields' field. response=%v", searchResult)})
			return
		}
		vt0 := vt.(map[string]interface{})
		vv := gin.H{
			"_index":  vt0["_index"],
			"_type":   vt0["_mapping"],
			"_id":     vt0["_id"],
			"_score":  nil,
			"_source": vt0,
			"sort":    []int64{0},
		}
		hitsES = append(hitsES, vv)
	}

	searchResultES := gin.H{
		"took":      tookES,
		"timed_out": false,
		"_shards": gin.H{
			"total":      5,
			"successful": 5,
			"skipped":    0,
			"failed":     0,
		},
		"hits": gin.H{
			"total":     len(hitsES),
			"max_score": nil,
			"hits":      hitsES,
		},
	}

	//ES RESPONSE {"took":24,"timed_out":false,"_shards":{"total":5,"successful":5,"skipped":0,"failed":0},"hits":{"total":3,"max_score":null,"hits":[{"_index":"conductor","_type":"workflow","_id":"9c545883-d76d-493e-a56c-5554ce1e5846","_score":null,"_source":{"workflowType":"ephemeralKitchenSinkEphemeralTasks","version":1,"workflowId":"9c545883-d76d-493e-a56c-5554ce1e5846","startTime":"2019-12-27T12:10:17.238Z","updateTime":"2019-12-27T12:10:17.286Z","status":"RUNNING","input":"{task2Name=task_10005}","output":"{}","executionTime":0,"inputSize":22,"outputSize":2},"sort":[1577448617238]},{"_index":"conductor","_type":"workflow","_id":"d3a4a1dc-2c56-46e5-b589-de4bfb024782","_score":null,"_source":{"workflowType":"ephemeralKitchenSinkStoredTasks","version":1,"workflowId":"d3a4a1dc-2c56-46e5-b589-de4bfb024782","startTime":"2019-12-27T12:10:17.162Z","updateTime":"2019-12-27T12:10:17.187Z","status":"RUNNING","input":"{task2Name=task_5}","output":"{}","executionTime":0,"inputSize":18,"outputSize":2},"sort":[1577448617162]},{"_index":"conductor","_type":"workflow","_id":"193f4a0f-00e0-4396-9d20-3d13e28ae7b3","_score":null,"_source":{"workflowType":"kitchensink","version":1,"workflowId":"193f4a0f-00e0-4396-9d20-3d13e28ae7b3","startTime":"2019-12-27T12:10:16.601Z","updateTime":"2019-12-27T12:10:17.027Z","status":"RUNNING","input":"{task2Name=task_5}","output":"{}","executionTime":0,"inputSize":18,"outputSize":2},"sort":[1577448616601]}]}}
	logrus.Debugf("Emulated ES response: %s", searchResultES)
	jsonWrite(w, http.StatusOK, searchResultES)
}

func processTerms(jsonValue []byte, blastQuery0 string) string {
	blastQuery := blastQuery0
	terms, _, _, err4 := jsonparser.Get(jsonValue, "terms")
	// /query/bool/must[]/terms
	if err4 != nil {
		logrus.Debugf("Couldn't find terms array in json=%s", err4)
		return blastQuery
	}
	// /query/bool/must[]/terms/
	err := jsonparser.ObjectEach(terms, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		// /query/bool/must[]/terms/fieldname
		// fmt.Printf("/query/bool/must[]/terms/fieldname Key: '%s'\n Value: '%s'\n Type: %s\n", string(key), string(value), dataType)

		_, err1 := jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			// fmt.Printf("/query/bool/must[]/terms/fieldname/[]fieldvalues Value: '%s'\n Type: %s\n", string(value), dataType)
			// /query/bool/must[]/terms/fieldname/[]fieldvalues
			//ADD THIS FIELD TO BLAST QUERY
			blastQuery = fmt.Sprintf("%s +%s:%s", blastQuery, strings.Trim(string(key), " "), string(value))
		})
		if err1 != nil {
			logrus.Debugf("Cannot parse terms array. err6=%s", err1)
		}
		return nil
	})
	if err != nil {
		logrus.Debugf("Cannot parse terms attribute. err=%s", err)
	}
	return blastQuery
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
