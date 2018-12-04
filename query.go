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
	"sync"
	"time"

	lexes "github.com/danlocke/lexes/parser"
	"fmt"
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

	// add query to database ...
	_, err = dbSaveQuery(i.db, req.Query, req.TopicId, auth, time.Now())
	if err != nil {
		return 500, err
	}

	docList := map[string]int{}
	for j := range req.Id {
		docList[req.Id[j]] = 0
	}

	log.Println("Query - ", req.Query)
	q, err := lexes.Parse(req.Query, "html", req.Fields, true, false)
	if err != nil {
		return 500, err
	}
	qry := *q
	qry["from"] = 0
	qry["size"] = i.config.Topics.PoolDepth * 2 

	buff, err := json.Marshal(qry)
	if err != nil {
		return 500, err
	}

	log.Printf("user %d - search - %s.\n", auth, req.Query)
	res, err := i.elasticSearchResponse(auth, strconv.FormatInt(req.TopicId, 10), buff)
	if err != nil {
		return 500, err
	}
	
	hits := []ApiCaseResponse{}
	count := 0

	// Exclude existing docIds from new results ...
	for j := range res.Results {
		if _, ok := docList[res.Results[j].Id]; !ok {
			hits = append(hits, res.Results[j])
			count++ 
			docList[res.Results[j].Id] = 0
		}
		if count >= i.config.Topics.PoolDepth {
			break
		}
		// add on a few extra docs if the pool depth have all been retrieved already...
		if j >= i.config.Topics.PoolDepth && count > 5 {
			break
		}
	}

	ret := TopicData{
		Queries: []queryRes{
			queryRes{
				Text: req.Query,
				Results: res.TotalHits,
				PooledResults: count,
			},
		},
		Results: hits,

	}

	wr, err := json.Marshal(ret)
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

func (i *Instance) elasticSearchResponse(userId int64, topicId string, query []byte) (*ApiSearchResponse, error) {
	esRes, err := i.es.Search(i.searchIndex, query, "")
	if err != nil {
		return nil, err
	}
	api, err := i.elasticSearchToApiSearchResponse(userId, topicId, esRes)
	return api, err
}

// func (i *Instance) elasticHits(userId int64, topicId string, queries []string) ([]ApiCaseResponse, error) {
// 	res := make([]ApiCaseResponse, 0)
// 	for q := range queries {
// 		qry, err := lexes.ParseJson(queries[q], "html", []string{"name", "date_filed", "id", "html"}, true, true)
// 		if err != nil {
// 			return nil, err
// 		}

// 		esRes, err := i.es.Search(i.searchIndex, qry)
// 		if err != nil {
// 			return nil, err
// 		}

// 		api, err := i.elasticSearchToApiCaseResponse(userId, topicId, esRes)
// 		if err != nil {
// 			return nil, err
// 		}
// 		res = append(res, api...)
// 	}
// 	return res, nil
// }

// Really just exclude duplicates over the set of queries...
// Returns total hits, pooled hits for each, result list, errors... 
func (i *Instance) elasticTopicQueryHits(userId int64, topicId string, queryStrings []string, queries []map[string]interface{}) ([]queryRes, []ApiCaseResponse, error) {
	retQueue :=  make(chan queryRes, len(queries))
	hitsQueue := make(chan ApiCaseResponse, i.config.Topics.PoolDepth * len(queries))

	seenId := new(sync.Map)
	errorChan := make(chan error, len(queries) * 3)

	var wg sync.WaitGroup

	for x := range queries {
		wg.Add(1)
		go func(x int) {
			defer wg.Done()
			q := queries[x]
			q["_source"] = []string{"id", "name"}
			q["from"] = 0
			q["size"] = i.config.Topics.PoolDepth
			qry, err := json.Marshal(q)
			if err != nil {
				errorChan <- err 	
			}

			esRes, err := i.es.Search(i.searchIndex, qry, "")
			if err != nil {
				errorChan <- err 	
			}

			api, err := i.elasticSearchToApiSearchResponse(userId, topicId, esRes)
			if err != nil {
				errorChan <- err 	
			}

			count := 0 
			for j := range api.Results {
				if _, ok := seenId.LoadOrStore(api.Results[j].Id, 0); !ok {
					hitsQueue <- api.Results[j]
					count++
				}
			}

			retQueue <- queryRes{
				Text: queryStrings[x],
				PooledResults: count, 
				Results: api.TotalHits,
			}

		}(x)
		
	}

	wg.Wait()

	close(errorChan)
	close(retQueue)
	close(hitsQueue)

	if len(errorChan) >= 1 {
		return nil, nil, <-errorChan
	}

	stats := []queryRes{}
	cases := []ApiCaseResponse{}

	for t := range retQueue {
		stats = append(stats, t)
	}

	for t := range hitsQueue {
		cases = append(cases, t)
	}

	return stats, cases, nil
}


func (i *Instance) elasticTopicDocListQuery(userId int64, topicId string) ([]queryRes, []ApiCaseResponse, error) {
	ids := i.docList[topicId]
	exclude := i.excludeList[topicId]
	query := map[string]interface{}{
		"filter": map[string]interface{}{
			"should": map[string]interface{}{
				"ids": map[string]interface{}{
					"values": ids,
				},
			},
			"must_not": map[string]interface{}{
				"ids": map[string]interface{}{
					"values": exclude,
				},
			},
		},
	}

	qry, err := json.Marshal(query)
	if err != nil {
		return nil, nil, err
	}

	fmt.Printf("Query: %s\n", qry)

	esRes, err := i.es.Search(i.searchIndex, qry, "")
	if err != nil {
		return nil, nil, err
	}

	api, err := i.elasticSearchToApiSearchResponse(userId, topicId, esRes)
	if err != nil {
		return nil, nil, err
	}

	return []queryRes{{
		Text: i.topics[topicId].Topic,
		Results: len(api.Results),
		PooledResults: len(api.Results),
	}}, api.Results, nil
}


// -----------------------------------------------------------------------------
// For creating standard match queries from pieces of text
func makeEsMatch(text, field string) map[string]interface{} {
	return map[string]interface{} {
		"query" : map[string]interface{} {
			"match" : map[string]interface{} {
				field : text,
			},
		},
	}
}

func createTextQuery(s, field string) map[string]interface{} {
	nums := getNumbers(s)
	text := cleanText(s)
	text = text + " " + nums
	return makeEsMatch(text, field)
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
