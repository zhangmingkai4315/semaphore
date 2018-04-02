package tasks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/zhangmingkai4315/semaphore/api/sockets"
	"github.com/zhangmingkai4315/semaphore/db"
)

func (t *task) log(msg string) {
	now := time.Now()

	for _, user := range t.users {
		b, err := json.Marshal(&map[string]interface{}{
			"type":       "log",
			"output":     msg,
			"time":       now,
			"task_id":    t.task.ID,
			"project_id": t.projectID,
		})

		if err != nil {
			panic(err)
		}

		sockets.Message(user, b)
	}

	go func() {
		_, err := db.Mysql.Exec("insert into task__output (task_id, task, output, time) VALUES (?, '', ?, ?)", t.task.ID, msg, now)
		if err != nil {
			fmt.Printf("Failed to insert task output: %s\n", err.Error())
			panic(err)
		}
	}()
}

func (t *task) updateStatus() {
	for _, user := range t.users {
		b, err := json.Marshal(&map[string]interface{}{
			"type":       "update",
			"start":      t.task.Start,
			"end":        t.task.End,
			"status":     t.task.Status,
			"task_id":    t.task.ID,
			"project_id": t.projectID,
		})

		if err != nil {
			panic(err)
		}

		sockets.Message(user, b)
	}

	if _, err := db.Mysql.Exec("update task set status=?, start=?, end=? where id=?", t.task.Status, t.task.Start, t.task.End, t.task.ID); err != nil {
		fmt.Printf("Failed to update task status: %s\n", err.Error())
		t.log("Fatal error with database!")
		panic(err)
	}
}

func Readln(r *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

func (t *task) logPipe(reader *bufio.Reader) {

	line, err := Readln(reader)
	for err == nil {
		t.log(line)
		line, err = Readln(reader)
	}

	if err != nil && err.Error() != "EOF" {
		//don't panic on this errors, sometimes it throw not dangerous "read |0: file already closed" error
		fmt.Printf("Failed to read task output: %s\n", err.Error())
	}

}

func (t *task) logCmd(cmd *exec.Cmd) {
	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()

	go t.logPipe(bufio.NewReader(stderr))
	go t.logPipe(bufio.NewReader(stdout))
}
