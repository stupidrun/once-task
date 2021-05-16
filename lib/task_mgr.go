package lib

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"log"
	"net/http"
	"sync"
	"time"
)

type DefaultTaskJob struct {
	taskId    cron.EntryID
	notifyUrl string
	param     string
	TimeStr   string
	finished  chan cron.EntryID
}

type TaskJob interface {
	cron.Job
	GetId() cron.EntryID
	String() string
	GetTime() string
}

func (t *DefaultTaskJob) innerRun() (*http.Response, error) {
	c := http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       time.Second * 5,
	}
	resp, err := c.Get(t.notifyUrl)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (t *DefaultTaskJob) Run() {
	log.Printf("task %d running: notifyUrl : %s ; params:  %s \n", t.taskId, t.notifyUrl, t.param)
	resp, err := t.innerRun()
	if err != nil {
		log.Println("task ", t.taskId, " run err:", err.Error())
	} else {
		log.Println("task finished with code: ", resp.StatusCode)
	}
	t.finished <- t.taskId
}

func (t *DefaultTaskJob) GetId() cron.EntryID {
	return t.taskId
}

func (t *DefaultTaskJob) String() string {
	return fmt.Sprintf("#Task %d -%s- -%s- @%s@ #", t.taskId, t.notifyUrl, t.param, t.TimeStr)
}

func (t *DefaultTaskJob) GetNotifyUrl() string {
	return t.notifyUrl
}

func (t *DefaultTaskJob) GetTime() string {
	return t.TimeStr
}

type TaskManager interface {
	AddTask(job *TaskJob) error
	RemoveTask(id cron.EntryID)
	TaskExists(id cron.EntryID) bool
	GetTasks() map[cron.EntryID]*TaskJob
	Start()
	Stop()
}

type DefaultTaskMgr struct {
	cron       *cron.Cron
	removeChan chan cron.EntryID
	tasks      map[cron.EntryID]TaskJob
	isRunning  bool
	lock       *sync.Mutex
}

func NewDefaultTaskMgr() *DefaultTaskMgr {
	return &DefaultTaskMgr{
		cron:       cron.New(),
		removeChan: make(chan cron.EntryID),
		tasks:      make(map[cron.EntryID]TaskJob),
		isRunning:  false,
		lock:       &sync.Mutex{},
	}
}

func (d *DefaultTaskMgr) GetTasks() map[cron.EntryID]TaskJob {
	return d.tasks
}

func (d *DefaultTaskMgr) addToCron(job TaskJob) (cron.EntryID, error) {
	location, _ := time.LoadLocation("Asia/Shanghai")
	t, err := time.ParseInLocation("20060102 15:04:05", job.GetTime(), location)
	if err != nil {
		return -1, err
	}
	id, err := d.cron.AddJob(fmt.Sprintf("%d %d %d %d *", t.Minute(), t.Hour(), t.Day(), t.Month()), job)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (d *DefaultTaskMgr) AddTask(notifyUrl string, param string, timeStr string) error {
	task := &DefaultTaskJob{
		notifyUrl: notifyUrl,
		param:     param,
		TimeStr:   timeStr,
		finished:  nil,
	}
	taskId, err := d.addToCron(task)
	if err != nil {
		log.Println("error: ", err.Error())
		return err
	}
	task.taskId = taskId
	task.finished = d.removeChan
	log.Printf("add task %v \n", task)
	log.Printf("task %d will running at %s", task.taskId, d.cron.Entry(task.taskId).Next)
	d.lock.Lock()
	defer d.lock.Unlock()
	d.tasks[taskId] = task
	return nil
}

func (d *DefaultTaskMgr) removeAllTask() {
	for k := range d.tasks {
		d.RemoveTask(k)
	}
}

func (d *DefaultTaskMgr) TaskExists(id cron.EntryID) bool {
	_, exists := d.tasks[id]
	return exists
}

func (d *DefaultTaskMgr) RemoveTask(id cron.EntryID) {
	if d.TaskExists(id) {
		log.Println("remove task ", id)
		d.lock.Lock()
		defer d.lock.Unlock()
		delete(d.tasks, id)
	}
}

func (d *DefaultTaskMgr) Start() {
	d.cron.Start()
	d.cron.Run()
	d.isRunning = true

	log.Println("TaskMgr start running")
	for {
		select {
		case taskId := <-d.removeChan:
			if d.TaskExists(taskId) {
				d.RemoveTask(taskId)
			}
		default:
			time.Sleep(time.Nanosecond * 50)
			if !d.isRunning {
				log.Println("TaskMgr stop")
				return
			}
		}
	}
}

func (d *DefaultTaskMgr) Stop() {
	d.cron.Stop()
	d.removeAllTask()
	d.isRunning = false
}
