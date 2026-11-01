package wittymodule

import (
	"errors"
	"fmt"
	"regexp"
)

// NotifyTask is the notification a helper sends to report work in the background, which
// the server shows with its own. A server names a task by the module it came from as well
// as its key, so a module can only ever speak about its own.
const NotifyTask = "task"

// A task's state. A process that ends has its running tasks ended for it.
const (
	TaskRunning = "running"
	TaskDone    = "done"
	TaskFailed  = "failed"
)

// TaskReport is one task's state as of now. Reports replace each other, so a server that
// drops some for coming too quickly loses nothing but steps along the way.
type TaskReport struct {
	Key   string `json:"key"`             // this module's name for it, which stays the same
	Title string `json:"title"`           // what it is, for people
	State string `json:"state"`           //
	Done  int64  `json:"done,omitempty"`  // how far along, with Total
	Total int64  `json:"total,omitempty"` // zero when nobody knows how long it will be
	Item  string `json:"item,omitempty"`  // what it's on now, such as a file's name
}

var taskKeyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// Validate checks a report. A server calls this on every one, since it comes from a helper.
func (t *TaskReport) Validate() error {
	switch {
	case !taskKeyRe.MatchString(t.Key):
		return fmt.Errorf("%q is not a task's key", t.Key)
	case t.Title == "" || !plainText(t.Title, 200, false):
		return errors.New("a task needs a title, in one line of plain text")
	case !plainText(t.Item, 200, false):
		return errors.New("what a task is on has to be one line of plain text")
	case t.State != TaskRunning && t.State != TaskDone && t.State != TaskFailed:
		return fmt.Errorf("%q is not a task's state", t.State)
	case t.Done < 0 || t.Total < 0 || (t.Total > 0 && t.Done > t.Total):
		return errors.New("a task's progress doesn't make sense")
	}
	return nil
}

// ReportTask tells the server how a task is going. Send it as things change, at most a few
// times a second.
func ReportTask(t TaskReport) error {
	if err := t.Validate(); err != nil {
		return err
	}
	return Notify(NotifyTask, t)
}
