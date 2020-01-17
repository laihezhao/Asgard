package controllers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"Asgard/client"
	"Asgard/models"
	"Asgard/services"
)

type JobController struct {
	jobService     *services.JobService
	agentService   *services.AgentService
	groupService   *services.GroupService
	moniterService *services.MonitorService
	archiveService *services.ArchiveService
}

func NewJobController() *JobController {
	return &JobController{
		jobService:     services.NewJobService(),
		agentService:   services.NewAgentService(),
		groupService:   services.NewGroupService(),
		moniterService: services.NewMonitorService(),
		archiveService: services.NewArchiveService(),
	}
}

func (c *JobController) formatJob(info *models.Job) map[string]interface{} {
	data := map[string]interface{}{
		"ID":        info.ID,
		"Name":      info.Name,
		"GroupID":   info.GroupID,
		"AgentID":   info.AgentID,
		"Dir":       info.Dir,
		"Program":   info.Program,
		"Args":      info.Args,
		"StdOut":    info.StdOut,
		"StdErr":    info.StdErr,
		"Spec":      info.Spec,
		"Timeout":   info.Timeout,
		"IsMonitor": info.IsMonitor,
		"Status":    info.Status,
	}
	group := c.groupService.GetGroupByID(info.GroupID)
	if group != nil {
		data["GroupName"] = group.Name
	} else {
		data["GroupName"] = ""
	}
	agent := c.agentService.GetAgentByID(info.AgentID)
	if agent != nil {
		data["AgentName"] = agent.IP + ":" + agent.Port
	} else {
		data["AgentName"] = ""
	}
	return data
}

func (c *JobController) List(ctx *gin.Context) {
	agent := DefaultInt(ctx, "agent", 0)
	page := DefaultInt(ctx, "page", 1)
	where := map[string]interface{}{}
	if agent != 0 {
		where["agent_id"] = agent
	}
	jobList, total := c.jobService.GetJobPageList(where, page, PageSize)
	if jobList == nil {
		APIError(ctx, "获取计划任务列表失败")
	}
	list := []map[string]interface{}{}
	for _, job := range jobList {
		list = append(list, c.formatJob(&job))
	}
	mpurl := "/job/list"
	if agent != 0 {
		mpurl = "/job/list?agent=" + strconv.Itoa(agent)
	}
	ctx.HTML(StatusOK, "job/list", gin.H{
		"Subtitle":   "计划任务列表",
		"List":       list,
		"Total":      total,
		"Pagination": PagerHtml(total, page, mpurl),
	})
}

func (c *JobController) Show(ctx *gin.Context) {
	id := DefaultInt(ctx, "id", 0)
	if id == 0 {
		JumpError(ctx)
		return
	}
	job := c.jobService.GetJobByID(int64(id))
	if job == nil {
		JumpError(ctx)
		return
	}
	ctx.HTML(StatusOK, "job/show", gin.H{
		"Subtitle": "查看计划任务",
		"Job":      c.formatJob(job),
	})
}

func (c *JobController) Monitor(ctx *gin.Context) {
	id := DefaultInt(ctx, "id", 0)
	if id == 0 {
		JumpError(ctx)
		return
	}
	cpus := []string{}
	memorys := []string{}
	times := []string{}
	moniters := c.moniterService.GetJobMonitor(id, 100)
	for _, moniter := range moniters {
		cpus = append(cpus, FormatFloat(moniter.CPU))
		memorys = append(memorys, FormatFloat(moniter.Memory))
		times = append(times, FormatTime(moniter.CreatedAt))
	}
	ctx.HTML(StatusOK, "app/monitor", gin.H{
		"Subtitle": "监控信息",
		"CPU":      cpus,
		"Memory":   memorys,
		"Time":     times,
	})
}

func (c *JobController) Archive(ctx *gin.Context) {
	id := DefaultInt(ctx, "id", 0)
	page := DefaultInt(ctx, "page", 1)
	where := map[string]interface{}{
		"type":       models.TYPE_JOB,
		"related_id": id,
	}
	if id == 0 {
		JumpError(ctx)
		return
	}
	archiveList, total := c.archiveService.GetArchivePageList(where, page, PageSize)
	if archiveList == nil {
		APIError(ctx, "获取归档列表失败")
	}
	list := []map[string]interface{}{}
	for _, archive := range archiveList {
		list = append(list, formatArchive(&archive))
	}
	mpurl := fmt.Sprintf("/job/archive?id=%d", id)
	ctx.HTML(StatusOK, "job/archive", gin.H{
		"Subtitle":   "归档列表",
		"List":       list,
		"Total":      total,
		"Pagination": PagerHtml(total, page, mpurl),
	})
}

func (c *JobController) Add(ctx *gin.Context) {
	ctx.HTML(StatusOK, "job/add", gin.H{
		"Subtitle":  "添加计划任务",
		"GroupList": c.groupService.GetUsageGroup(),
		"AgentList": c.agentService.GetUsageAgent(),
	})
}

func (c *JobController) Create(ctx *gin.Context) {
	groupID := FormDefaultInt(ctx, "group_id", 0)
	name := ctx.PostForm("name")
	agentID := FormDefaultInt(ctx, "agent_id", 0)
	dir := ctx.PostForm("dir")
	program := ctx.PostForm("program")
	args := ctx.PostForm("args")
	stdOut := ctx.PostForm("std_out")
	stdErr := ctx.PostForm("std_err")
	spec := ctx.PostForm("spec")
	timeout := FormDefaultInt(ctx, "timeout", 0)
	isMonitor := ctx.PostForm("is_monitor")
	if !Required(ctx, &name, "名称不能为空") {
		return
	}
	if !Required(ctx, &dir, "执行目录不能为空") {
		return
	}
	if !Required(ctx, &program, "执行程序不能为空") {
		return
	}
	if !Required(ctx, &stdOut, "标准输出路径不能为空") {
		return
	}
	if !Required(ctx, &stdErr, "错误输出路径不能为空") {
		return
	}
	if !Required(ctx, &spec, "运行配置不能为空") {
		return
	}
	if agentID == 0 {
		APIBadRequest(ctx, "运行实例不能为空")
		return
	}
	job := new(models.Job)
	job.GroupID = int64(groupID)
	job.Name = name
	job.AgentID = int64(agentID)
	job.Dir = dir
	job.Program = program
	job.Args = args
	job.StdOut = stdOut
	job.StdErr = stdErr
	job.Spec = spec
	job.Timeout = int64(timeout)
	job.Status = 0
	job.Creator = GetUserID(ctx)
	if isMonitor != "" {
		job.IsMonitor = 1
	}
	ok := c.jobService.CreateJob(job)
	if !ok {
		APIError(ctx, "创建计划任务失败")
		return
	}
	APIOK(ctx)
}

func (c *JobController) Edit(ctx *gin.Context) {
	id := DefaultInt(ctx, "id", 0)
	if id == 0 {
		JumpError(ctx)
		return
	}
	job := c.jobService.GetJobByID(int64(id))
	if job == nil {
		JumpError(ctx)
		return
	}
	ctx.HTML(StatusOK, "job/edit", gin.H{
		"Subtitle":  "编辑计划任务",
		"Job":       c.formatJob(job),
		"GroupList": c.groupService.GetUsageGroup(),
		"AgentList": c.agentService.GetUsageAgent(),
	})
}

func (c *JobController) Update(ctx *gin.Context) {
	id := FormDefaultInt(ctx, "id", 0)
	groupID := FormDefaultInt(ctx, "group_id", 0)
	name := ctx.PostForm("name")
	agentID := FormDefaultInt(ctx, "agent_id", 0)
	dir := ctx.PostForm("dir")
	program := ctx.PostForm("program")
	args := ctx.PostForm("args")
	stdOut := ctx.PostForm("std_out")
	stdErr := ctx.PostForm("std_err")
	spec := ctx.PostForm("spec")
	timeout := FormDefaultInt(ctx, "timeout", 0)
	isMonitor := ctx.PostForm("is_monitor")
	if id == 0 {
		APIBadRequest(ctx, "ID格式错误")
		return
	}
	if !Required(ctx, &name, "名称不能为空") {
		return
	}
	if !Required(ctx, &dir, "执行目录不能为空") {
		return
	}
	if !Required(ctx, &program, "执行程序不能为空") {
		return
	}
	if !Required(ctx, &stdOut, "标准输出路径不能为空") {
		return
	}
	if !Required(ctx, &stdErr, "错误输出路径不能为空") {
		return
	}
	if !Required(ctx, &spec, "运行配置不能为空") {
		return
	}
	if agentID == 0 {
		APIBadRequest(ctx, "运行实例不能为空")
		return
	}
	job := c.jobService.GetJobByID(int64(id))
	if job == nil {
		APIBadRequest(ctx, "计划任务不存在")
		return
	}
	job.GroupID = int64(groupID)
	job.Name = name
	job.AgentID = int64(agentID)
	job.Dir = dir
	job.Program = program
	job.Args = args
	job.StdOut = stdOut
	job.StdErr = stdErr
	job.Spec = spec
	job.Timeout = int64(timeout)
	job.Updator = GetUserID(ctx)
	if isMonitor != "" {
		job.IsMonitor = 1
	}
	ok := c.jobService.UpdateJob(job)
	if !ok {
		APIError(ctx, "更新计划任务失败")
		return
	}
	APIOK(ctx)
}

func (c *JobController) Delete(ctx *gin.Context) {
	id := FormDefaultInt(ctx, "id", 0)
	if id == 0 {
		APIBadRequest(ctx, "ID格式错误")
		return
	}
	job := c.jobService.GetJobByID(int64(id))
	if job == nil {
		APIError(ctx, "计划任务不存在")
		return
	}
	if job.Status == 1 {
		APIError(ctx, "计划任务正在运行不能删除")
		return
	}
	job.Status = -1
	job.Updator = GetUserID(ctx)
	ok := c.jobService.UpdateJob(job)
	if !ok {
		APIError(ctx, "删除应用失败")
		return
	}
	APIOK(ctx)
}

func (c *JobController) Start(ctx *gin.Context) {
	id := DefaultInt(ctx, "id", 0)
	if id == 0 {
		APIBadRequest(ctx, "ID格式错误")
		return
	}
	job := c.jobService.GetJobByID(int64(id))
	if job == nil {
		APIError(ctx, "计划任务不存在")
		return
	}
	if job.Status == 1 {
		APIError(ctx, "计划任务已经启动")
		return
	}
	agent := c.agentService.GetAgentByID(job.AgentID)
	if agent == nil {
		APIError(ctx, "计划任务对应实例获取异常")
		return
	}
	_job, err := client.GetAgentJob(agent, int64(id))
	if err != nil {
		APIError(ctx, fmt.Sprintf("获取计划任务情况异常:%s", err.Error()))
		return
	}
	if _job == nil {
		err = client.AddAgentJob(agent, job)
		if err != nil {
			APIError(ctx, fmt.Sprintf("添加计划任务异常:%s", err.Error()))
			return
		}
		job.Status = 1
		c.jobService.UpdateJob(job)
		APIOK(ctx)
		return
	}
	job.Status = 1
	c.jobService.UpdateJob(job)
	APIOK(ctx)
}

func (c *JobController) ReStart(ctx *gin.Context) {
	id := DefaultInt(ctx, "id", 0)
	if id == 0 {
		APIBadRequest(ctx, "ID格式错误")
		return
	}
	job := c.jobService.GetJobByID(int64(id))
	if job == nil {
		APIError(ctx, "计划任务不存在")
		return
	}
	agent := c.agentService.GetAgentByID(job.AgentID)
	if agent == nil {
		APIError(ctx, "计划任务对应实例获取异常")
		return
	}
	_job, err := client.GetAgentJob(agent, int64(id))
	if err != nil {
		APIError(ctx, fmt.Sprintf("获取计划任务情况异常:%s", err.Error()))
		return
	}
	if _job == nil {
		err = client.AddAgentJob(agent, job)
		if err != nil {
			APIError(ctx, fmt.Sprintf("重启异常:%s", err.Error()))
			return
		}
		APIOK(ctx)
	}
	err = client.UpdateAgentJob(agent, job)
	if err != nil {
		APIError(ctx, fmt.Sprintf("重启异常:%s", err.Error()))
		return
	}
	APIOK(ctx)
}

func (c *JobController) Pause(ctx *gin.Context) {
	id := DefaultInt(ctx, "id", 0)
	if id == 0 {
		APIBadRequest(ctx, "ID格式错误")
		return
	}
	job := c.jobService.GetJobByID(int64(id))
	if job == nil {
		APIError(ctx, "计划任务不存在")
		return
	}
	agent := c.agentService.GetAgentByID(job.AgentID)
	if agent == nil {
		APIError(ctx, "计划任务对应实例获取异常")
		return
	}
	_job, err := client.GetAgentJob(agent, int64(id))
	if err != nil {
		APIError(ctx, fmt.Sprintf("获取计划任务情况异常:%s", err.Error()))
		return
	}
	if _job == nil {
		APIOK(ctx)
		return
	}
	err = client.RemoveAgentJob(agent, int64(id))
	if err != nil {
		APIError(ctx, fmt.Sprintf("停止计划任务异常:%s", err.Error()))
		return
	}
	job.Status = 2
	job.Updator = GetUserID(ctx)
	c.jobService.UpdateJob(job)
	APIOK(ctx)
}