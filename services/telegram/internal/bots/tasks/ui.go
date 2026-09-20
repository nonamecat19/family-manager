package tasks

import (
	"context"
	"strconv"
	"time"

	"connectrpc.com/connect"

	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/telegram/internal/bot"
	"github.com/nnc/family-manager/services/telegram/internal/i18n"
)

func (c *client) home(ctx context.Context, cc *bot.Context) (string, bot.Keyboard, error) {
	res, err := c.todayTasks(ctx, cc)
	if err != nil {
		return "", nil, err
	}

	text := bot.Lines(
		bot.Bold(cc.T(i18n.TasksTitle)),
		"",
		cc.T(i18n.TasksToday)+"   "+bot.Bold(strconv.Itoa(len(res))),
		"",
		bot.Italic(cc.T(i18n.TasksPickDone)),
	)

	keyboard := bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.HelpTasksAdd), "add"), bot.Data(cc.T(i18n.HelpTasksToday), "today")),
		bot.Row(bot.Data(cc.T(i18n.HelpTasksMine), "mine"), bot.Data(cc.T(i18n.HelpTasksDone), "done")),
	}
	return text, keyboard, nil
}

func (c *client) todayTasks(ctx context.Context, cc *bot.Context) ([]*tasksv1.Task, error) {
	req := connect.NewRequest(&tasksv1.ListTasksRequest{
		Filter: tasksv1.TaskFilter_TASK_FILTER_OPEN,
	})
	cc.Authorize(req)

	res, err := c.rpc.ListTasks(ctx, req)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var filtered []*tasksv1.Task
	for _, t := range res.Msg.GetTasks() {
		if t.GetDueOn() != "" {
			dueDate, err := time.Parse(time.DateOnly, t.GetDueOn())
			if err == nil && !dueDate.After(now) {
				filtered = append(filtered, t)
			}
		}
	}
	return filtered, nil
}