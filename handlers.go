package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type TopicData struct {

	Queries []string

	Results []ApiCaseResponse

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

	if id < 1 {
		http.Redirect(w, r, redirectTarget, 302)
		// return 200, nil
	} else {
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
	i.templates["index"].Execute(w, r)
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
	queries = append(queries, topic.EsQuery...)
	for _, s := range []string{topic.CitingSentence, topic.CitingParagraph}{
		queries = append(queries, createTextQuery(s))
	}

	qry, err := dbGetUserQueries(i.db, topicId, auth)
	if err != nil {
		return 500, err
	}
	for q := range qry {
		queries = append(queries, makeEsMatch(qry[q]))
	}

	hits, err := i.elasticTopicQueryHits(auth, topicId, queries)
	if err != nil {
		return 500, err
	}

	assessed, err := dbGetAssessedPerTopic(i.db, auth, topicId)
	if err != nil {
		return 500, err
	}

	for j := range hits {
		hits[j].Stored = false
		if k, ok := assessed[hits[j].Id]; ok {
			hits[j].Relevance = k
			hits[j].Stored = true
		}
	}

	queryString := []string{}
	queryString = append(queryString, topic.Query...)
	queryString = append(queryString, qry...)

	t := TopicData {
		Queries: queryString,
		Results: hits,
	}

	buff, err := json.Marshal(t)
	if err != nil {
		return 500, err
	}
	w.Write(buff)
	return 200, nil
}
