package main

import (
	"database/sql"
	"net/http"
)

func (i *Instance) authed(r *http.Request) (int64, error) {
	session, err := i.store.Get(r, "assess")
	if err != nil {
		return int64(-1), err
	}

	val, ok := session.Values["id"]
	if !ok {
		return int64(-1), nil
	}
	return val.(int64), nil
}

func dbGetUserId(db *sql.DB, name, pass string) (int64, error) {
	var rows *sql.Rows
	var err error
	rows, err = db.Query("SELECT user_id FROM users WHERE name = $1 AND pass = $2",
		name, pass)
	if err != nil {
		return 0, err
	}

	defer rows.Close()
	for rows.Next() {
		var user_id int64
		err := rows.Scan(&user_id)
		if err != nil {
			return 0, err
		}
		return user_id, nil
	}
	return -1, nil
}
