package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	lexes "lexes/parser"
)

type topicSearchPostReq struct {

	Query string `json:"query"`

	TopicId int64 `json:"topic"`

	Fields []string  `json:"fields"`

	Id []string `json:"ids"`

}

var textRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)
var numRe = regexp.MustCompile(`[0-9]+`)

func apiSearch(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		return 401, errors.New("Unauthorized")
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return 500, err
	}
	var req topicSearchPostReq

	err = json.Unmarshal(body, &req)
	if err != nil {
		return 500, err
	}

	docList := map[string]int{}
	for j := range req.Id {
		docList[req.Id[j]] = 0
	}

	qry, err := lexes.ParseJson(req.Query, "html", req.Fields, true, false)
	if err != nil {
		return 500, err
	}

	// add query to database ...
	_, err = dbSaveQuery(i.db, req.Query, req.TopicId, auth, time.Now())
	if err != nil {
		return 500, err
	}

	log.Printf("user %d - search - %s.\n", auth, req.Query)
	res, err := i.elasticSearchResponse(auth, strconv.FormatInt(req.TopicId, 10), qry)
	if err != nil {
		return 500, err
	}
	ret := []ApiCaseResponse{}
	// Exclude existing docIds from new results ...
	for j := range res {
		if _, ok := docList[res[j].Id]; !ok {
			docList[res[j].Id] = 0
			ret = append(ret, res[j])
		}
	}

	wr, err := json.Marshal(res)
	if err != nil {
		return 500, err
	}

	w.Write(wr)
	return 200, nil
}

func dbSaveQuery(db *sql.DB, query string, topic, user int64, date time.Time) (sql.Result, error) {
	return db.Exec("INSERT INTO query (topic_id, query, user_id, date_added) VALUES ($1, $2, $3, $4)",
		topic, query, user, date)
}

func dbGetUserQueries(db *sql.DB, topic string, user int64) ([]string, error) {
	rows, err := db.Query("SELECT query FROM query WHERE topic_id = $1 AND user_id = $2",
		topic, user)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	queries := make([]string, 0)
	for rows.Next() {
		var query string
		err := rows.Scan(&query)
		if err != nil {
			return nil, err
		}
		queries = append(queries, query)
	}
	return queries, nil
}

func (i *Instance) elasticSearchResponse(userId int64, topicId string, query []byte) ([]ApiCaseResponse, error) {
	esRes, err := i.es.Search(i.searchIndex, query)
	if err != nil {
		return nil, err
	}
	api, err := i.elasticSearchToApiCaseResponse(userId, topicId, esRes)
	return api, err
}

func (i *Instance) elasticHits(userId int64, topicId string, queries []string) ([]ApiCaseResponse, error) {
	res := make([]ApiCaseResponse, 0)
	for q := range queries {
		qry, err := lexes.ParseJson(queries[q], "html", []string{"case_name", "date_filed", "id", "html"}, true, true)
		if err != nil {
			return nil, err
		}

		esRes, err := i.es.Search(i.searchIndex, qry)
		if err != nil {
			return nil, err
		}

		api, err := i.elasticSearchToApiCaseResponse(userId, topicId, esRes)
		if err != nil {
			return nil, err
		}
		res = append(res, api...)
	}
	return res, nil
}

func (i *Instance) elasticTopicQueryHits(userId int64, topicId string, queries []map[string]interface{}) ([]ApiCaseResponse, error) {
	res := make([]ApiCaseResponse, 0)
	seenId := map[string]int{}
	for _, q := range queries {
		q["_source"] = []string{"id", "case_name"}
		q["from"] = 0
		q["size"] = 30
		qry, err := json.Marshal(q)
		if err != nil {
			return nil, err
		}
		esRes, err := i.es.Search(i.searchIndex, qry)
		if err != nil {
			return nil, err
		}

		api, err := i.elasticSearchToApiCaseResponse(userId, topicId, esRes)
		if err != nil {
			return nil, err
		}
		for j := range api {
			if _, ok := seenId[api[j].Id]; !ok {
				res = append(res, api[j])
			}
			seenId[api[j].Id] = 0
		}
	}
	return res, nil
}

// -----------------------------------------------------------------------------
// For creating standard match queries from pieces of text
func makeEsMatch(s string) map[string]interface{} {
	return map[string]interface{} {
		"query" : map[string]interface{} {
			"plain_text" : s,
		},
		"from" : 0,
		"size" : 30,
	}
}

func createTextQuery(s string) map[string]interface{} {
	nums := getNumbers(s)
	text := cleanText(s)
	text = text + " " + nums
	q := map[string]interface{} {
		"query" : map[string]interface{} {
			"match" : map[string]interface{} {
				"plain_text" : text,
			},
		},
	}

	return q
}

func cleanText(text string) string {
	return strings.ToLower(textRe.ReplaceAllString(text, " "))
}

func getNumbers(text string) string {
	nums := numRe.FindAllString(text, -1)
	var ret bytes.Buffer
	seen := make(map[string]int)
	for _, num := range nums {
		if _, ok := seen[string(num)]; !ok {
			ret.WriteString(num)
			ret.WriteString(" ")
			seen[string(num)] = 0
		}
	}

	return strings.Trim(ret.String(), " ")
}


// func (i *Instance) getNumResultsForManualQueries() error {
// 	for k, v := range i.topics {
// 		numRes := 0
// 		for q := range v.Query {
// 			qry, err := json.Marshal(v.EsQuery[q])
// 			if err != nil {
// 				return err
// 			}
// 			esRes, err := i.es.Count(i.searchIndex, qry)
// 			if err != nil {
// 				log.Panic(err)
// 				return err
// 			}
//
// 			numRes += int(esRes.Count)
// 		}
// 		v.NumResults = numRes
// 		i.topics[k] = v
// 	}
//
// 	return nil
// }
