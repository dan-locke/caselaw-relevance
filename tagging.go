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

	tag_id SERIAL UNIQUE,

	topic_id bigint,

	doc_id bigint,

	tagger int,

	date_added TIMESTAMP,

	start_pos bigint NOT NULL,

	end_pos bigint NOT NULL,

	start_offset bigint NOT NULL,

	end_offset bigint NOT NULL,

	start_container VARCHAR NOT NULL,

	end_container VARCHAR NOT NULL,

	PRIMARY KEY (topic_id, doc_id, tagger, date_added),

	FOREIGN KEY (tagger) REFERENCES users (user_id)

);*/

type Tag struct {

	TopicId int64 `json:"-"`

	DocId int64 `json:"doc_id"`

	UserId int64 `json:"-"`

	Date time.Time

	Start int64 `json:"start"`

	End int64 `json:"end"`

	StartOffset int64 `json:"start_offset"`

	EndOffset int64 `json:"end_offset"`

	StartContainer string `json:"start_container"`

	EndContainer string `json:"end_container"`

}

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
	rows, err = db.Query("SELECT doc_id, date_added, start_offset, end_offset, start_container, end_container FROM tag WHERE topic_id = $1 AND doc_id = $2 AND tagger = $3",
		topicId, docId, userId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	tags := make([]Tag, 0)
	for rows.Next() {
		var doc_id int64
		var date_added time.Time
		var start_offset int64
		var end_offset int64
		var start_container string
		var end_container string

		err := rows.Scan(&doc_id, &date_added, &start_offset, &end_offset,
			&start_container, &end_container)
		if err != nil {
			return nil, err
		}
		tags = append(tags, Tag{
			DocId: doc_id,
			Date: date_added,
			StartOffset: start_offset,
			EndOffset: end_offset,
			StartContainer: start_container,
			EndContainer: end_container,
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

	insertId, err := dbSaveTag(i.db, tag)
	if err != nil {
		return 500, err
	}
	log.Println("InsertId -", insertId)
	buff, err := json.Marshal(struct{ Id int `json:"id"`}{Id: insertId}, )
	if err != nil {
		return 500, err
	}

	log.Printf("user %d - saving tag - %d.\n", auth, tag.DocId)
	w.Write(buff)
	return 200, nil
}

func dbSaveTag(db *sql.DB, t Tag) (int, error) {
	var tag_id int
	err := db.QueryRow("INSERT INTO tag (topic_id, doc_id, tagger, date_added, start_pos, end_pos, start_offset, end_offset, start_container, end_container) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING tag_id",
		t.TopicId, t.DocId, t.UserId, t.Date, t.Start, t.End, t.StartOffset,
			t.EndOffset, t.StartContainer, t.EndContainer).Scan(&tag_id)

	return tag_id, err
}


// Delete tag ------------------------------------------------------------------

func apiDeleteTag(i *Instance, w http.ResponseWriter, r *http.Request) (int, error) {
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

	var tag struct{ Id int `json:"id"`}
	err = json.Unmarshal(body, &tag)
	_, err = dbDeleteTag(i.db, tag.Id)
	if err != nil {
		return 500, err
	}
	log.Printf("user %d - deleting tag - %d.\n", auth, tag.Id)
	return 200, nil
}

func dbDeleteTag(db *sql.DB, tagId int) (sql.Result, error) {
	return db.Exec("DELETE FROM tag WHERE tag_id = $1", tagId)
}
