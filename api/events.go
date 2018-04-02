package api

import (
	"net/http"

	"github.com/castawaylabs/mulekick"
	"github.com/gorilla/context"
	"github.com/masterminds/squirrel"
	"github.com/zhangmingkai4315/semaphore/db"
)

func getEvents(w http.ResponseWriter, r *http.Request, limit uint64) {
	user := context.Get(r, "user").(*db.User)

	q := squirrel.Select("event.*, p.name as project_name").
		From("event").
		LeftJoin("project as p on event.project_id=p.id").
		OrderBy("created desc")

	if limit > 0 {
		q = q.Limit(limit)
	}

	projectObj, exists := context.GetOk(r, "project")
	if exists == true {
		// limit query to project
		project := projectObj.(db.Project)
		q = q.Where("event.project_id=?", project.ID)
	} else {
		q = q.LeftJoin("project__user as pu on pu.project_id=p.id").
			Where("p.id IS NULL or pu.user_id=?", user.ID)
	}

	var events []db.Event

	query, args, _ := q.ToSql()
	if _, err := db.Mysql.Select(&events, query, args...); err != nil {
		panic(err)
	}

	for i, evt := range events {
		if evt.ObjectID == nil || evt.ObjectType == nil {
			continue
		}

		var q squirrel.SelectBuilder

		switch *evt.ObjectType {
		case "task":
			q = squirrel.Select("case when length(task.playbook) > 0 then task.playbook else tpl.playbook end").
				From("task").
				Join("project__template as tpl on task.template_id=tpl.id").
				Where("task.id=?", evt.ObjectID)
		default:
			continue
		}

		query, args, _ := q.ToSql()
		name, err := db.Mysql.SelectNullStr(query, args...)
		if err != nil {
			panic(err)
		}

		if name.Valid == true {
			events[i].ObjectName = name.String
		}
	}

	mulekick.WriteJSON(w, http.StatusOK, events)
}

func getLastEvents(w http.ResponseWriter, r *http.Request) {
	getEvents(w, r, 200)
}

func getAllEvents(w http.ResponseWriter, r *http.Request) {
	getEvents(w, r, 0)
}
