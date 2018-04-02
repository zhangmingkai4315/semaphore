package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/context"
	"github.com/zhangmingkai4315/semaphore/db"
	"github.com/zhangmingkai4315/semaphore/util"
)

func authentication(w http.ResponseWriter, r *http.Request) {
	var userID int

	if authHeader := strings.ToLower(r.Header.Get("authorization")); len(authHeader) > 0 && strings.Contains(authHeader, "bearer") {
		var token db.APIToken
		if err := db.Mysql.SelectOne(&token, "select * from user__token where id=? and expired=0", strings.Replace(authHeader, "bearer ", "", 1)); err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			panic(err)
		}

		userID = token.UserID
	} else {
		// fetch session from cookie
		cookie, err := r.Cookie("semaphore")
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		value := make(map[string]interface{})
		if err = util.Cookie.Decode("semaphore", cookie.Value, &value); err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		user, ok := value["user"]
		sessionVal, okSession := value["session"]
		if !ok || !okSession {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		userID = user.(int)
		sessionID := sessionVal.(int)

		// fetch session
		var session db.Session
		if err := db.Mysql.SelectOne(&session, "select * from session where id=? and user_id=? and expired=0", sessionID, userID); err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if time.Now().Sub(session.LastActive).Hours() > 7*24 {
			// more than week old unused session
			// destroy.
			if _, err := db.Mysql.Exec("update session set expired=1 where id=?", sessionID); err != nil {
				panic(err)
			}

			w.WriteHeader(http.StatusForbidden)
			return
		}

		if _, err := db.Mysql.Exec("update session set last_active=UTC_TIMESTAMP() where id=?", sessionID); err != nil {
			panic(err)
		}
	}

	user, err := db.FetchUser(userID)
	if err != nil {
		fmt.Println("Can't find user", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	context.Set(r, "user", user)
}
