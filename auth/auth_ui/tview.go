//go:build ignore

package auth_ui

import (
	"errors"
	"io"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rusq/dlog"
)

type TView struct {
	app *tview.Application

	mustStop      chan struct{}
	inputReceived chan struct{}
	done          chan struct{}
}

func (tv *TView) RequestWorkspace(w io.Writer) (string, error) {
	tv.inputReceived = make(chan struct{}, 1)
	tv.mustStop = make(chan struct{}, 1)
	tv.done = make(chan struct{}, 1)
	tv.app = tview.NewApplication()

	var workspace string
	var exit bool
	input := tview.NewInputField().SetLabel("Slack Workspace").SetFieldWidth(40)
	form := tview.NewForm().
		AddFormItem(input).
		AddButton("OK", func() {
			workspace = input.GetText()
			tv.wait()
		}).
		AddButton("Cancel", func() {
			exit = true
			tv.wait()
		})

	form.SetBorder(true).
		SetTitle(" Slackdump EZ-Login 3000 ").
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if !input.HasFocus() {
				return event
			}
			switch event.Key() {
			default:
				return event
			case tcell.KeyCR:
				workspace = input.GetText()
			case tcell.KeyESC:
				exit = true

			}
			tv.wait()
			return nil
		})

	go func() {
		if err := tv.app.SetRoot(modal(form, 60, 7), true).EnableMouse(true).Run(); err != nil {
			dlog.Println(err)
		}
	}()

	// waiting for the user to finish interaction
	<-tv.inputReceived
	if exit {
		tv.app.Stop()
		return "", errors.New("operation cancelled")
	}
	return workspace, nil
}

func (tv *TView) wait() {
	close(tv.inputReceived)
	<-tv.mustStop
	tv.app.Stop()
	close(tv.done)
}

func (tv *TView) Stop() {
	close(tv.mustStop)
	<-tv.done
}

func modal(p tview.Primitive, width int, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
