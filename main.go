package main

import (
	"flag"
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"stupid.run/tasks/lib"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 9099, "listen port")
	taskMgr := lib.NewDefaultTaskMgr()
	go taskMgr.Start()

	web := iris.New()
	web.Use(logger.New())
	web.Use(recover.New())

	web.Post("/task", func(ctx iris.Context) {
		notifyUrl := ctx.PostValueTrim("notify_url")
		paramStr := ctx.PostValueTrim("param_str")
		timeStr := ctx.PostValueTrim("time_str")
		if notifyUrl == "" || paramStr == "" || timeStr == "" {
			ctx.StatusCode(500)
			ctx.JSON(iris.Map{
				"err": "Invalid params",
			})
			return
		}
		if err := taskMgr.AddTask(notifyUrl, paramStr, timeStr); err != nil {
			web.Logger().Println(err.Error())
			ctx.StatusCode(500)
			ctx.JSON(iris.Map{
				"err": err.Error(),
			})
			return
		}
		ctx.StatusCode(201)
		ctx.JSON(iris.Map{
			"err": "ok",
		})
	})

	web.Get("/", func(ctx iris.Context) {
		tasks := taskMgr.GetTasks()
		result := make([]string, len(tasks))
		for _, v := range tasks {
			result = append(result, v.String())
		}
		ctx.JSON(result)
		return
	})

	web.Run(iris.Addr(fmt.Sprintf(":%d", port)))
	taskMgr.Stop()
}
