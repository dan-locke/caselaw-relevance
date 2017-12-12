package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Assessment struct {

	TopicId int64

	DocId int64

	UserId int64

	Date time.Time

	Relevance string

}

type assessmentBodyRequest struct {

	Assessments []struct {

		Id       int64 `json:"id"`

		Relevance string   `json:"relevance"`

	} `json:"assessments"`

	Id int64 `json:"id"`

}

func apiAssessTopic(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
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
		log.Panic(err)
		return 400, err
	}
	var res assessmentBodyRequest
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Panic(err)
		return 400, err
	}

	date := time.Now()

	for j := range res.Assessments {
		dbSaveTopicAssessment(i.db, Assessment{
			TopicId: res.Id,
			DocId: res.Assessments[j].Id,
			UserId: auth,
			Relevance: res.Assessments[j].Relevance,
			Date: date,
		})
	}
	return 200, nil
}

func dbSaveTopicAssessment(db *sql.DB, a Assessment) (sql.Result, error) {
	return db.Exec("INSERT INTO assessment (doc_id, topic_id, assessor, relevant, date_assessed) VALUES ($1, $2, $3, $4, $5)",
		a.DocId, a.TopicId, a.UserId, a.Relevance, a.Date)
}

func dbGetNumberAssessedPerTopic(db *sql.DB, user int64) (map[string]int, error) {
	rows, err := db.Query("SELECT topic_id, COUNT(DISTINCT doc_id) FROM assessment WHERE assessor = $1 GROUP BY topic_id", user)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	assessed := map[string]int{}
	for rows.Next() {
		var topic_id string
		var count int
		err = rows.Scan(&topic_id, &count)
		if err != nil {
			return nil, err
		}
		assessed[topic_id] = count
	}
	return assessed, nil
}

func dbGetAssessedPerTopic(db *sql.DB, user int64, topicId string) (map[string]string, error) {
	rows, err := db.Query("SELECT doc_id, relevant FROM assessment WHERE assessor = $1 AND topic_id = $2",
		user, topicId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	assessed := map[string]string{}
	for rows.Next() {
		var doc_id string
		var relevance string
		err = rows.Scan(&doc_id, &relevance)
		if err != nil {
			return nil, err
		}
		assessed[doc_id] = relevance
	}
	return assessed, nil
}

// func dbGetNumberAssessedTopics(db *sql.DB, user string) (int, error) {
// 	rows, err := db.Query("SELECT COUNT(topic_id) FROM assessment WHERE assessor = $1", user)
// 	if err != nil {
// 		return 0, err
// 	}
//
// 	defer rows.Close()
// 	var count int
// 	for rows.Next() {
// 		err = rows.Scan(&count)
// 		if err != nil {
// 			return 0, err
// 		}
// 	}
// 	return count, nil
// }
//
// func dbGetAssessedTopics(db *sql.DB, user string) (map[string]bool, error) {
// 	rows, err := db.Query("SELECT topic_id FROM assessment WHERE assessor = $1", user)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	defer rows.Close()
// 	assessed := map[string]bool{}
// 	for rows.Next() {
// 		var topic_id string
// 		err = rows.Scan(&topic_id)
// 		if err != nil {
// 			return nil, err
// 		}
// 		assessed[topic_id] = true
// 	}
// 	return assessed, nil
// }
//
// func dbIsAssessedTopic(db *sql.DB, user, topicId string) (bool, error) {
// 	rows, err := db.Query("SELECT topic_id FROM assessment WHERE assessor = $1 AND topic_id = $2", user, topicId)
// 	if err != nil {
// 		return false, err
// 	}
//
// 	defer rows.Close()
// 	var topic_id string
// 	for rows.Next() {
// 		err = rows.Scan(&topic_id)
// 		if err != nil {
// 			return false, err
// 		}
// 	}
// 	if topicId == topic_id {
// 		return true, nil
// 	}
// 	return false, nil
// }
