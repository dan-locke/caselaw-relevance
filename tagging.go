package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"log"
	"time"

	"github.com/gorilla/mux"
)

/* Tags are described in the databse as follows:

CREATE TABLE tag (

	topic_id bigint,

	doc_id bigint,

	tagger int,

	date_added TIMESTAMP,

	start_pos bigint NOT NULL,

	end_pos bigint NOT NULL,

	PRIMARY KEY (topic_id, doc_id, tagger, date_added),

	FOREIGN KEY (tagger) REFERENCES users (user_id)

);*/

type Tag struct {

	TopicId int64 `json:"topic_id"`

	DocId int64 `json:"doc_id"`

	UserId int64

	Date time.Time

	Start int64 `json:"start"`

	End int64 `json:"end"`

}

// type Tag struct {
//
// 	TopicId int64 `json:"topic_id"` `pq:"topic_id"`
//
// 	DocId int64 `json:"doc_id"` `pq:"doc_id"`
//
// 	UserId int64 `pq:"tagger"`
//
// 	Date time.Time `pq:"date_added"`
//
// 	Start int64 `json:"start"` `pq:"start_pos"`
//
// 	End int64 `json:"end"` `pq:"end_pos"`
//
// }

func getTagHandler(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
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
	docId, ok := vars["docId"]; if !ok {
		return 400, nil
	}

	log.Printf("user %d - getting tags for %s - %s.\n", auth, topicId, docId)

	tags, err := dbGetTags(i.db, topicId, docId, auth)
	if err != nil {
		return 500, err
	}
	wr, err := json.Marshal(tags)
	if err != nil {
		return 500, err
	}

	w.Write(wr)
	return 200, nil
}

func dbGetTags(db *sql.DB, topicId, docId string, userId int64) ([]Tag, error) {
	var rows *sql.Rows
	var err error
	rows, err = db.Query("SELECT doc_id, date_added, start_pos, end_pos FROM tag WHERE topic_id = $1 AND doc_id = $2 AND tagger = $3",
		topicId, docId, userId)
	if err != nil {
		return []Tag{}, err
	}

	defer rows.Close()
	tags := make([]Tag, 0)
	for rows.Next() {
		var doc_id int64
		var date_added time.Time
		var start_pos int64
		var end_pos int64
		err := rows.Scan(&doc_id, &date_added, &start_pos, &end_pos)
		if err != nil {
			return nil, err
		}
		tags = append(tags, Tag{
			DocId: doc_id,
			Date: date_added,
			Start: start_pos,
			End: end_pos,
		})
	}
	return tags, nil
}

// Save tags -------------------------------------------------------------------

func apiSaveTag(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
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

	var tag Tag
	err = json.Unmarshal(body, &tag)
	if err != nil {
		return 500, err
	}

	tag.UserId = auth
	tag.Date = time.Now()

	_, err = dbSaveTag(i.db, tag)
	if err != nil {
		return 500, err
	}

	log.Printf("user %d - saving tag - %d.\n", auth, tag.DocId)
	return 200, nil
}

func dbSaveTag(db *sql.DB, t Tag) (sql.Result, error) {
	return db.Exec("INSERT INTO tag (topic_id, doc_id, tagger, date_added, start_pos, end_pos) VALUES ($1, $2, $3, $4, $5, $6)",
		t.TopicId, t.DocId, t.UserId, t.Date, t.Start, t.End)
}
