package tasks

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/sdk/go/tasks/v1/tasksv1connect"
	"github.com/nnc/family-manager/services/telegram/internal/bot"
	"github.com/nnc/family-manager/services/telegram/internal/i18n"
)

const intro = string(i18n.TasksIntro)

type client struct {
	rpc tasksv1connect.TasksServiceClient
}

func Bot(httpClient *http.Client, addr string) bot.Options {
	c := &client{rpc: tasksv1connect.NewTasksServiceClient(httpClient, addr)}

	return bot.Options{
		Intro: intro,
		Home:  c.home,
		Commands: []bot.Command{
			{Name: "add", Args: "[title] [date] [time]", Help: string(i18n.HelpTasksAdd), Run: c.add},
			{Name: "today", Help: string(i18n.HelpTasksToday), Run: c.today},
			{Name: "mine", Help: string(i18n.HelpTasksMine), Run: c.mine},
			{Name: "done", Help: string(i18n.HelpTasksDone), Run: c.done},
		},
		Callbacks: []bot.Callback{
			{Prefix: "tdone", Run: c.donePick},
		},
		OnText: c.onText,
	}
}

func (c *client) add(ctx context.Context, cc *bot.Context) error {
	if cc.Args == "" {
		return cc.Reply(ctx, cc.T(i18n.TasksAddUsage))
	}

	parts := strings.SplitN(cc.Args, " ", 3)
	title := parts[0]
	var dueOn, dueTime string

	if len(parts) >= 2 {
		dueOn = parts[1]
	}
	if len(parts) >= 3 {
		dueTime = parts[2]
	}

	if dueOn != "" {
		dueOn = parseDate(cc, dueOn)
		if dueOn == "" {
			return bot.Invalid("%s", cc.T(i18n.TasksBadDate))
		}
	}

	req := connect.NewRequest(&tasksv1.CreateTaskRequest{
		Title:   title,
		DueOn:   dueOn,
		DueTime: dueTime,
	})
	cc.Authorize(req)

	res, err := c.rpc.CreateTask(ctx, req)
	if err != nil {
		return err
	}

	return cc.Reply(ctx, cc.T(i18n.TasksAdded, bot.Esc(res.Msg.GetTask().GetTitle())))
}

func (c *client) today(ctx context.Context, cc *bot.Context) error {
	res, err := c.list(ctx, cc, &tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_OPEN,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	var filtered []*tasksv1.Task
	for _, t := range res {
		if t.GetDueOn() != "" {
			dueDate, err := time.Parse(time.DateOnly, t.GetDueOn())
			if err == nil && !dueDate.After(now) {
				filtered = append(filtered, t)
			}
		}
	}
	return c.showTasks(ctx, cc, filtered, i18n.TasksToday)
}

func (c *client) mine(ctx context.Context, cc *bot.Context) error {
	res, err := c.list(ctx, cc, &tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_MINE,
	})
	if err != nil {
		return err
	}
	return c.showTasks(ctx, cc, res, i18n.TasksMine)
}

func (c *client) done(ctx context.Context, cc *bot.Context) error {
	res, err := c.list(ctx, cc, &tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_MINE,
	})
	if err != nil {
		return err
	}

	if len(res) == 0 {
		return cc.Reply(ctx, cc.T(i18n.TasksNothingMine))
	}

	var buttons []bot.Button
	for _, t := range res {
		if t.GetStatus() == tasksv1.TaskStatus_TASK_STATUS_OPEN {
			buttons = append(buttons, bot.Data(bot.Esc(t.GetTitle()), fmt.Sprintf("tdone:%s", t.GetId())))
		}
	}

	if len(buttons) == 0 {
		return cc.Reply(ctx, cc.T(i18n.TasksNothingOpen))
	}

	keyboard := bot.Keyboard{}.Grid(buttons, 1)
	keyboard = append(keyboard, bot.Row(bot.Data(cc.T(i18n.Back), "home")))

	return cc.Send(ctx, bot.Lines(
		bot.Bold(cc.T(i18n.TasksPickDone)),
		"",
	), keyboard)
}

func (c *client) donePick(ctx context.Context, cc *bot.Context) error {
	taskID := cc.Payload()
	if taskID == "" {
		return bot.Invalid("%s", cc.T(i18n.TasksGone))
	}

	req := connect.NewRequest(&tasksv1.CompleteTaskRequest{
		TaskId: taskID,
	})
	cc.Authorize(req)

	_, err := c.rpc.CompleteTask(ctx, req)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bot.Invalid("%s", cc.T(i18n.TasksGone))
		}
		if connect.CodeOf(err) == connect.CodeFailedPrecondition {
			return bot.Invalid("%s", cc.T(i18n.TasksAlreadyDone))
		}
		return err
	}

	return cc.Show(ctx, bot.Lines(
		bot.Bold(cc.T(i18n.TasksMarkedDone)),
		"",
		cc.T(i18n.TasksDoneOK),
	), bot.Keyboard{bot.Row(bot.Data(cc.T(i18n.Menu), "home"))})
}

func (c *client) list(ctx context.Context, cc *bot.Context, req *tasksv1.ListTasksRequest) ([]*tasksv1.Task, error) {
	connReq := connect.NewRequest(req)
	cc.Authorize(connReq)

	res, err := c.rpc.ListTasks(ctx, connReq)
	if err != nil {
		return nil, err
	}
	return res.Msg.GetTasks(), nil
}

func (c *client) showTasks(ctx context.Context, cc *bot.Context, tasks []*tasksv1.Task, titleKey i18n.Key) error {
	if len(tasks) == 0 {
		var nothingKey i18n.Key
		switch titleKey {
		case i18n.TasksToday:
			nothingKey = i18n.TasksNothingToday
		case i18n.TasksMine:
			nothingKey = i18n.TasksNothingMine
		default:
			nothingKey = i18n.TasksNothingOpen
		}
		return cc.Reply(ctx, cc.T(nothingKey))
	}

	lines := []string{bot.Bold(cc.T(titleKey)), ""}
	for _, t := range tasks {
		line := bot.Bold(bot.Esc(t.GetTitle()))
		if t.GetDueOn() != "" {
			dueTime := ""
			if t.GetDueTime() != "" {
				dueTime = " " + t.GetDueTime()[:5]
			}
			line += "\n      " + cc.T(i18n.TasksDueOn, t.GetDueOn()+dueTime)
		} else {
			line += "\n      " + cc.T(i18n.TasksNoDeadline)
		}
		priority := formatPriority(cc, t.GetPriority())
		line += " · " + priority
		lines = append(lines, line)
	}

	return cc.Send(ctx, strings.Join(lines, "\n"), bot.Keyboard{bot.Row(bot.Data(cc.T(i18n.Menu), "home"))})
}

func (c *client) onText(ctx context.Context, cc *bot.Context) error {
	return cc.Reply(ctx, bot.Italic(cc.T(i18n.OnlyButtons)))
}

func parseDate(cc *bot.Context, raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	now := time.Now()

	switch raw {
	case "today", "сьогодні":
		return now.Format(time.DateOnly)
	case "tomorrow", "завтра":
		return now.AddDate(0, 0, 1).Format(time.DateOnly)
	}

	if _, err := time.Parse(time.DateOnly, raw); err == nil {
		return raw
	}

	return ""
}

func formatPriority(cc *bot.Context, p tasksv1.Priority) string {
	switch p {
	case tasksv1.Priority_PRIORITY_LOW:
		return cc.T(i18n.TasksPriorityLow)
	case tasksv1.Priority_PRIORITY_MEDIUM:
		return cc.T(i18n.TasksPriorityMedium)
	case tasksv1.Priority_PRIORITY_HIGH:
		return cc.T(i18n.TasksPriorityHigh)
	case tasksv1.Priority_PRIORITY_URGENT:
		return cc.T(i18n.TasksPriorityUrgent)
	default:
		return ""
	}
}