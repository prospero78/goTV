// +build !darwin

package tv

import (
	"github.com/atotto/clipboard"
	xs "github.com/huandu/xstrings"
	term "github.com/nsf/termbox-go"

	"github.com/prospero78/goTV/tv/autowidth"
	"github.com/prospero78/goTV/tv/types"
)

/*
TEditField is a single-line text edit contol. Edit field consumes some keyboard
events when it is active: all printable charaters; Delete, BackSpace, Home,
End, left and right arrows; Ctrl+R to clear TEditField.
Edit text can be limited. By default a user can enter text of any length.
Use SetMaxWidth to limit the maximum text length. If the text is longer than
maximun then the text is automatically truncated.
TEditField calls onChage in case of its text is changed. Event field Msg contains the new text
*/
type TEditField struct {
	TBaseControl
	// cursor position in edit text
	cursorPos types.ACoordX
	// the number of the first displayed text character - it is used in case of text is longer than edit width
	offset    int
	readonly  bool
	maxWidth  int
	showStars bool

	onChange   func(Event)
	onKeyPress func(term.Key, rune) bool

	autoWidth types.IAutoWidth
}

// NewEditField creates a new EditField control
// view - is a View that manages the control
// parent - is container that keeps the control. The same View can be a view and a parent at the same time.
// width - is minimal width of the control.
// text - text to edit.
// scale - the way of scaling the control when the parent is resized. Use DoNotScale constant if the
//  control should keep its original size.
func CreateEditField(parent IControl, width int, text string, scale int) *TEditField {
	e := &TEditField{
		TBaseControl: NewBaseControl(),
		autoWidth:    autowidth.New(),
	}

	e.onChange = nil
	e.SetTitle(text)
	e.SetEnabled(true)

	if width == 0 {
		e.autoWidth.Set()
		width = xs.Len(text) + 1
	}

	e.SetSize(width, 1)
	e.cursorPos = types.ACoordX(xs.Len(text))
	e.offset = 0
	e.parent = parent
	e.readonly = false
	e.SetScale(scale)

	e.SetConstraints(width, 1)

	e.End()

	if parent != nil {
		parent.AddChild(e)
	}

	return e
}

/*
ProcessEvent processes all events come from the control parent. If a control
processes an event it should return true. If the method returns false it means
that the control do not want or cannot process the event and the caller sends
the event to the control parent
*/
func (e *TEditField) ProcessEvent(event Event) bool {
	if !e.Active() || !e.Enabled() {
		return false
	}

	if event.Type == EventActivate && event.X == 0 {
		term.HideCursor()
	}

	if event.Type == EventKey && event.Key != term.KeyTab {
		if e.onKeyPress != nil {
			res := e.onKeyPress(event.Key, event.Ch)
			if res {
				return true
			}
		}

		switch event.Key {
		case term.KeyEnter:
			return false
		case term.KeySpace:
			e.InsertRune(' ')
			return true
		case term.KeyBackspace, term.KeyBackspace2:
			e.Backspace()
			return true
		case term.KeyDelete:
			e.Del()
			return true
		case term.KeyArrowLeft:
			e.CharLeft()
			return true
		case term.KeyHome:
			e.Home()
			return true
		case term.KeyEnd:
			e.End()
			return true
		case term.KeyCtrlR:
			if !e.readonly {
				e.Clear()
			}
			return true
		case term.KeyArrowRight:
			e.CharRight()
			return true
		case term.KeyCtrlC:
			if !e.showStars {
				_ = clipboard.WriteAll(e.Title())
			}
			return true
		case term.KeyCtrlV:
			if !e.readonly {
				s, _ := clipboard.ReadAll()
				e.SetTitle(s)
				e.End()
			}
			return true
		default:
			if event.Ch != 0 {
				e.InsertRune(event.Ch)
				return true
			}
		}
		return false
	}

	return false
}
