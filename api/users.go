package api

import (
	"database/sql"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/castawaylabs/mulekick"
	"github.com/gorilla/context"
	"github.com/zhangmingkai4315/semaphore/db"
	"github.com/zhangmingkai4315/semaphore/util"
	"golang.org/x/crypto/bcrypt"
)

func getUsers(w http.ResponseWriter, r *http.Request) {
	var users []db.User
	if _, err := db.Mysql.Select(&users, "select * from user"); err != nil {
		panic(err)
	}

	mulekick.WriteJSON(w, http.StatusOK, users)
}

func addUser(w http.ResponseWriter, r *http.Request) {
	var user db.User
	if err := mulekick.Bind(w, r, &user); err != nil {
		return
	}

	editor := context.Get(r, "user").(*db.User)
	if editor.Admin != true {
		log.Warn(editor.Username + " is not permitted to create users")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user.Created = time.Now()

	if err := db.Mysql.Insert(&user); err != nil {
		panic(err)
	}

	mulekick.WriteJSON(w, http.StatusCreated, user)
}

func getUserMiddleware(w http.ResponseWriter, r *http.Request) {
	userID, err := util.GetIntParam("user_id", w, r)
	if err != nil {
		return
	}

	var user db.User
	if err := db.Mysql.SelectOne(&user, "select * from user where id=?", userID); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		panic(err)
	}

	editor := context.Get(r, "user").(*db.User)
	if editor.Admin != true && editor.ID != user.ID {
		log.Warn(editor.Username + " is not permitted to edit users")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	context.Set(r, "_user", user)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	oldUser := context.Get(r, "_user").(db.User)
	editor := context.Get(r, "user").(*db.User)

	var user db.User
	if err := mulekick.Bind(w, r, &user); err != nil {
		return
	}

	if editor.Admin != true && editor.ID != oldUser.ID {
		log.Warn(editor.Username + " is not permitted to edit users")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if editor.ID == oldUser.ID && oldUser.Admin != user.Admin {
		log.Warn("User can't edit his own role")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if oldUser.External == true && oldUser.Username != user.Username {
		log.Warn("Username is not editable for external LDAP users")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, err := db.Mysql.Exec("update user set name=?, username=?, email=?, alert=?, admin=? where id=?", user.Name, user.Username, user.Email, user.Alert, user.Admin, oldUser.ID); err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func updateUserPassword(w http.ResponseWriter, r *http.Request) {
	user := context.Get(r, "_user").(db.User)
	editor := context.Get(r, "user").(*db.User)

	var pwd struct {
		Pwd string `json:"password"`
	}

	if editor.Admin != true && editor.ID != user.ID {
		log.Warn(editor.Username + " is not permitted to edit users")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if user.External == true {
		log.Warn("Password is not editable for external LDAP users")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := mulekick.Bind(w, r, &pwd); err != nil {
		return
	}

	password, _ := bcrypt.GenerateFromPassword([]byte(pwd.Pwd), 11)
	if _, err := db.Mysql.Exec("update user set password=? where id=?", string(password), user.ID); err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	user := context.Get(r, "_user").(db.User)
	editor := context.Get(r, "user").(*db.User)

	if editor.Admin != true && editor.ID != user.ID {
		log.Warn(editor.Username + " is not permitted to delete users")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if _, err := db.Mysql.Exec("delete from project__user where user_id=?", user.ID); err != nil {
		panic(err)
	}
	if _, err := db.Mysql.Exec("delete from user where id=?", user.ID); err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusNoContent)
}
