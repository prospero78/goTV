package tv

import (
	term "github.com/nsf/termbox-go"
	"github.com/prospero78/goTV/tv/cons"
	"github.com/prospero78/goTV/tv/widgets/event"
)

// Composer is a service object that manages Views and console, processes
// events, and provides service methods. One application must have only
// one object of this type
type mainLoop struct {
	// a channel to communicate with View(e.g, Views send redraw event to this channel)
	channel chan event.TEvent
}

var (
	loop *mainLoop
)

func initMainLoop() {
	loop = new(mainLoop)
	loop.channel = make(chan event.TEvent)
}

// MainLoop starts the main application event loop
func MainLoop() {
	RefreshScreen()

	eventQueue := make(chan term.Event)
	go func() {
		for {
			eventQueue <- term.PollEvent()
		}
	}()

	for {
		RefreshScreen()

		select {
		case ev := <-eventQueue:
			switch ev.Type {
			case term.EventError:
				panic(ev.Err)
			default:
				ProcessEvent(termboxEventToLocal(ev))
			}
		case cmd := <-loop.channel:
			if cmd.Type == cons.EventQuit {
				return
			}
			ProcessEvent(cmd)
		}
	}
}

func _putEvent(ev event.TEvent) {
	loop.channel <- ev
}

// PutEvent send event to a Composer directly.
// Used by Views to ask for repainting or for quitting the application
func PutEvent(ev event.TEvent) {
	go _putEvent(ev)
}
