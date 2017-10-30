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

	start_id INT NOT NULL,

	end_id INT NOT NULL,

	PRIMARY KEY (topic_id, doc_id, tagger, date_added),

	FOREIGN KEY (tagger) REFERENCES users (user_id)

);*/

type Tag struct {

	TagId int `json:"tag_id,omitempty"`

	TopicId int64 `json:"topic_id,omitempty"`

	DocId int64 `json:"doc_id"`

	UserId int64 `json:"-"`

	Date time.Time 	`json:"-"`

	Start int64 `json:"start"`

	End int64 `json:"end"`

	StartOffset int64 `json:"start_offset"`

	EndOffset int64 `json:"end_offset"`

	StartContainer string `json:"start_container"`

	EndContainer string `json:"end_container"`

	StartId int `json:"start_id"`

	EndId int `json:"end_id"`

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
	rows, err = db.Query("SELECT tag_id, doc_id, start_offset, end_offset, start_container, end_container, start_id, end_id FROM tag WHERE topic_id = $1 AND doc_id = $2 AND tagger = $3",
		topicId, docId, userId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	tags := make([]Tag, 0)
	for rows.Next() {
		var tag_id int
		var doc_id int64
		var start_offset int64
		var end_offset int64
		var start_container string
		var end_container string
		var start_id int
		var end_id int

		err := rows.Scan(&tag_id, &doc_id, &start_offset, &end_offset,
			&start_container, &end_container, &start_id, &end_id)
		if err != nil {
			return nil, err
		}
		tags = append(tags, Tag{
			TagId: tag_id,
			DocId: doc_id,
			StartOffset: start_offset,
			EndOffset: end_offset,
			StartContainer: start_container,
			EndContainer: end_container,
			StartId: start_id,
			EndId: end_id,
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
	err := db.QueryRow("INSERT INTO tag (topic_id, doc_id, tagger, date_added, start_pos, end_pos, start_offset, end_offset, start_container, end_container, start_id, end_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING tag_id",
		t.TopicId, t.DocId, t.UserId, t.Date, t.Start, t.End, t.StartOffset,
			t.EndOffset, t.StartContainer, t.EndContainer, t.StartId, t.EndId).Scan(&tag_id)

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
