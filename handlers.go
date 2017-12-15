package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	lexes "lexes/parser"
)

type TopicData struct {

	Queries []queryRes

	Results []ApiCaseResponse

}

type queryRes struct {

	Text string

	Results int

	PooledResults int
}

func loginViewHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	i.templates["login"].Execute(w, r)
	return 200, nil
}

func loginHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	name := r.FormValue("nm")
	name = strings.ToLower(name)
	pass := r.FormValue("pwd")
	log.Printf("login attempt for user %s.\n", name)

	id, err := dbGetUserId(i.db, name, pass)
	if err != nil {
		return 400, err
	}

	redirectTarget := "/login"

	if id > 0 {
		session, err := i.store.Get(r, "assess")
		if err != nil {
			return 500, err
		}
		session.Options.Path = "/"
		session.Values["name"] = name
		session.Values["id"] = id
		err = session.Save(r, w)
		if err != nil {
			return 500, err
		}
		redirectTarget = "/"
		log.Printf("user %s - logged in.\n", name)
	}
	http.Redirect(w, r, redirectTarget, 302)
	return 200, nil
}

func indexViewHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		http.Redirect(w, r, "/login", 302)
		// return 401, errors.New("Unauthorized")
	}
	log.Printf("user %d - handling index.\n", auth)
	http.Redirect(w, r, "/topics", 302)
	// i.templates["index"].Execute(w, r)
	return 200, nil
}

func infoViewHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		http.Redirect(w, r, "/login", 302)
		// return 401, errors.New("Unauthorized")
	}
	log.Printf("user %d - handling info.", auth)
	i.templates["info"].Execute(w, r)
	return 200, nil
}

func decisionViewHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		http.Redirect(w, r, "/login", 302)
	}
	vars := mux.Vars(r)
	docId, ok := vars["docId"]; if !ok {
		return 400, nil
	}
	log.Printf("user %d - handling decision - %s", auth, docId)

	i.templates["decision"].Execute(w, nil)
	return 200, nil
}

func decisionHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		http.Redirect(w, r, "/login", 302)
	}
	vars := mux.Vars(r)
	docId, ok := vars["docId"]; if !ok {
		return 400, nil
	}
	log.Printf("user %d - handling decision data - %s", auth, docId)

	getRes, err := i.es.Get(i.searchIndex, i.docType, docId)
	if err != nil {
		return 500, err
	}
	api, err := elasticGetToApiResponse(getRes)
	if err != nil {
		return 500, err
	}
	buff, err := json.Marshal(api)
	if err != nil {
		return 500, err
	}
	w.Write(buff)
	return 200, nil
}

func topicIndexViewHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		http.Redirect(w, r, "/login", 302)
		// return 401, errors.New("Unauthorized")
	}
	log.Printf("user %d - handling topic index.", auth)

	i.templates["topicIndex"].Execute(w, r)
	return 200, nil
}

func topicIndexDataHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		return 401, errors.New("Unauthorized")
	}
	log.Printf("user %d - handling topic index data.", auth)

	list, err := i.getTopicList(auth)
	if err != nil {
		return 500, err
	}
	buff, err := json.Marshal(list)
	if err != nil {
		return 500, err
	}
	w.Write(buff)
	return 200, nil
}

func topicViewHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		http.Redirect(w, r, "/login", 302)
		// return 401, errors.New("Unauthorized")
	}
	vars := mux.Vars(r)
	topicId, ok := vars["topicId"]; if !ok {
		return 400, nil
	}

	topic := i.getTopic(topicId)

	log.Printf("user %d - handling topic - %s.\n", auth, topicId)
	i.templates["topic"].Execute(w, topic)
	return 200, nil
}

func topicDecisionHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		http.Redirect(w, r, "/login", 302)
	}
	vars := mux.Vars(r)
	docId, ok := vars["docId"]; if !ok {
		return 400, nil
	}
	topicId, ok := vars["topicId"]; if !ok {
		return 400, nil
	}
	log.Printf("user %d - requested decision data for topic - %s", auth, topicId)

	getRes, err := i.es.Get(i.searchIndex, i.docType, docId)
	if err != nil {
		return 500, err
	}
	api, err := elasticGetToApiResponse(getRes)
	if err != nil {
		return 500, err
	}
	assessed, err := dbGetAssessedPerTopic(i.db, auth, topicId)
	if err != nil {
		return 500, err
	}

	api.Stored = false
	if a, ok := assessed[docId]; ok {
		api.Relevance = a
		api.Stored = true
	}

	buff, err := json.Marshal(api)
	if err != nil {
		return 500, err
	}
	w.Write(buff)
	return 200, nil
}

func topicDataHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	auth, err := i.authed(r)
	if err != nil {
		return 500, err
	}
	if auth < 0 {
		return 401, errors.New("Unauthorized")
	}
	vars := mux.Vars(r)
	topicId, ok := vars["topicId"]; if !ok {
		return 400, nil
	}
	log.Printf("user %d - requested topic data - %s.\n", auth, topicId)

	topic := i.getTopic(topicId)

	queries := make([]map[string]interface{}, 0)

	queries = append(queries, createTextQuery(topic.Topic, "html"))
	queryString := []string{topic.Topic}

	for _, e := range topic.Extracts {
		for _, q := range []string{e.CitingSentence, e.CitingParagraph}{ // will need to change this to fix for new topic struct... 
			queries = append(queries, createTextQuery(q, "html"))
			queryString = append(queryString, q)
		}

		queries = append(queries, e.EsQuery...)
		queryString = append(queryString, e.Query...)
	}

	qry, err := dbGetUserQueries(i.db, topicId, auth)
	if err != nil {
		return 500, err
	}

	for _, q := range qry {
		lq, err := lexes.Parse(q, "html", nil, true, false)
		if err != nil {
			return 500, err
		}
		queries = append(queries, *lq)
		queryString = append(queryString, q)
	}

	qStats, hits, err := i.elasticTopicQueryHits(auth, topicId, queries)
	if err != nil {
		return 500, err
	}

	t := TopicData {
		Queries: make([]queryRes, len(qStats)),
		Results: hits,
	}

	log.Println("Queries -", len(queryString))
	log.Println("Stats -", qStats)
	// log.Println(len(totals))
	// log.Println(len(pooled))
	// log.Println(len(queries))

	for j := range qStats {
		t.Queries[j].Text = queryString[j]
		t.Queries[j].Results = qStats[j].total
		t.Queries[j].PooledResults = qStats[j].count
	}

	log.Println("Gere")

	buff, err := json.Marshal(t)
	if err != nil {
		return 500, err
	}
	log.Println("here")

	w.Write(buff)
	return 200, nil
}
