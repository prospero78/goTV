package tv

import (
	"bufio"
	"os"
	"strings"

	xs "github.com/huandu/xstrings"
	term "github.com/nsf/termbox-go"

	"github.com/prospero78/goTV/tv/autoheight"
	"github.com/prospero78/goTV/tv/autowidth"
	"github.com/prospero78/goTV/tv/types"
)

/*
TextView is control to display a read-only text. Text can be
loaded from a file or set manually. A portions of text can be
added on the fly and if the autoscroll is enabled the control
scroll down to the end - it may be useful to create a log
viewer.
Content is scrollable with arrow keys or by clicking buttons
on the scrolls(a control can have up to 2 scrollbars: vertical
and horizontal. The latter one is available only if WordWrap
mode is off).
*/
type TextView struct {
	TBaseControl
	// own listbox members
	lines   []string
	lengths []int
	// for up/down scroll
	topLine int
	// for side scroll
	leftShift int
	wordWrap  bool
	// colorized     bool
	virtualHeight int
	virtualWidth  int
	autoscroll    bool
	maxLines      int

	autoWidth  types.IAutoWidth
	autoHeight types.IAutoHeight
}

/*
CreateTextView creates a new frame.
view - is a View that manages the control
parent - is container that keeps the control. The same View can be a view and a parent at the same time.
width and height - are minimal size of the control.
scale - the way of scaling the control when the parent is resized. Use DoNotScale constant if the
control should keep its original size.
*/
func CreateTextView(parent IControl, width, height int, scale int) *TextView {
	l := &TextView{
		TBaseControl: NewBaseControl(),
		autoWidth:    autowidth.New(),
		autoHeight:   autoheight.New(),
	}

	if height == 0 {
		height = 3
		l.autoWidth.Set()
	}
	if width == 0 {
		width = 5
		l.autoHeight.Set()
	}

	l.SetSize(width, height)
	l.SetConstraints(width, height)
	l.topLine = 0
	l.lines = make([]string, 0)
	l.parent = parent
	l.maxLines = 0

	l.SetTabStop(true)
	l.SetScale(scale)

	if parent != nil {
		parent.AddChild(l)
	}

	return l
}

func (l *TextView) outputHeight() int {
	h := l.height
	if !l.wordWrap {
		h.Set(h.Get() - 1)
	}
	return int(h.Get())
}

func (l *TextView) drawScrolls() {
	height := l.outputHeight()
	pos := ThumbPosition(l.topLine, l.virtualHeight-l.outputHeight(), height)
	DrawScrollBar(l.pos.GetX()+types.ACoordX(l.width.Get()-1), l.pos.GetY(), 1, height, pos)

	if !l.wordWrap {
		pos = ThumbPosition(l.leftShift, l.virtualWidth-int(l.width.Get())+1, int(l.width.Get())-1)
		DrawScrollBar(l.pos.GetX(), l.pos.GetY()+types.ACoordY(l.height.Get()-1), int(l.width.Get())-1, 1, pos)
	}
}

func (l *TextView) drawText() {
	PushAttributes()
	defer PopAttributes()

	maxWidth := l.width.Get() - 1
	maxHeight := l.outputHeight()

	bg, fg := RealColor(l.bg, l.Style(), ColorEditBack), RealColor(l.fg, l.Style(), ColorEditText)
	if l.Active() {
		bg, fg = RealColor(l.bg, l.Style(), ColorEditActiveBack), RealColor(l.fg, l.Style(), ColorEditActiveText)
	}

	SetTextColor(fg)
	SetBackColor(bg)
	if l.wordWrap {
		lineID := l.posToItemNo(l.topLine)
		linePos := l.itemNoToPos(lineID)

		y := 0
		for {
			if y >= maxHeight || lineID >= len(l.lines) {
				break
			}

			remained := l.lengths[lineID]
			start := 0
			for remained > 0 {
				s := SliceColorized(l.lines[lineID], start, start+int(maxWidth))

				if linePos >= l.topLine {
					DrawText(l.pos.GetX(), l.pos.GetY()+types.ACoordY(y), s)
				}

				remained -= int(maxWidth)
				y++
				linePos++
				start += int(maxWidth)

				if y >= maxHeight {
					break
				}
			}

			lineID++
		}
	} else {
		y := 0
		total := len(l.lines)
		for {
			if y+l.topLine >= total {
				break
			}
			if y >= maxHeight {
				break
			}

			str := l.lines[l.topLine+y]
			lineLength := l.lengths[l.topLine+y]
			if l.leftShift == 0 {
				if lineLength > int(maxWidth) {
					str = SliceColorized(str, 0, int(maxWidth))
				}
			} else {
				if l.leftShift+int(maxWidth) >= lineLength {
					str = SliceColorized(str, l.leftShift, -1)
				} else {
					str = SliceColorized(str, l.leftShift, int(maxWidth)+l.leftShift)
				}
			}
			DrawText(l.pos.GetX(), l.pos.GetY()+types.ACoordY(y), str)

			y++
		}
	}
}

// Repaint draws the control on its View surface
func (l *TextView) Draw() {
	if l.hidden {
		return
	}

	PushAttributes()
	defer PopAttributes()

	x, y := l.pos.Get()
	w, h := l.Size()

	bg, fg := RealColor(l.bg, l.Style(), ColorEditBack), RealColor(l.fg, l.Style(), ColorEditText)
	if l.Active() {
		bg, fg = RealColor(l.bg, l.Style(), ColorEditActiveBack), RealColor(l.fg, l.Style(), ColorEditActiveText)
	}

	SetTextColor(fg)
	SetBackColor(bg)
	FillRect(x, y, w, h, ' ')
	l.drawText()
	l.drawScrolls()
}

func (l *TextView) home() {
	l.topLine = 0
}

func (l *TextView) End() {
	height := l.outputHeight()

	if l.virtualHeight <= height {
		return
	}

	if l.topLine+height >= l.virtualHeight {
		return
	}

	l.topLine = l.virtualHeight - height
}

func (l *TextView) moveUp(dy int) {
	if l.topLine == 0 {
		return
	}

	if l.topLine <= dy {
		l.topLine = 0
	} else {
		l.topLine -= dy
	}
}

func (l *TextView) moveDown(dy int) {
	end := l.topLine + l.outputHeight()

	if end >= l.virtualHeight {
		return
	}

	if l.topLine+dy+l.outputHeight() >= l.virtualHeight {
		l.topLine = l.virtualHeight - l.outputHeight()
	} else {
		l.topLine += dy
	}
}

func (l *TextView) moveLeft() {
	if l.wordWrap || l.leftShift == 0 {
		return
	}

	l.leftShift--
}

func (l *TextView) moveRight() {
	if l.wordWrap {
		return
	}

	if l.leftShift+int(l.width.Get())-1 >= l.virtualWidth {
		return
	}

	l.leftShift++
}

func (l *TextView) processMouseClick(ev Event) bool {
	if ev.Key != term.MouseLeft {
		return false
	}

	dx := ev.X - l.pos.GetX()
	dy := ev.Y - l.pos.GetY()
	yy := l.outputHeight()

	// cursor is not on any scrollbar
	if int(dx) != int(l.width.Get()-1) && int(dy) != int(l.height.Get())-1 {
		return false
	}
	// wordwrap mode does not have horizontal scroll
	if l.wordWrap &&
		int(dx) != int(l.width.Get())-1 {
		return false
	}
	// corner in not wordwrap mode
	if !l.wordWrap &&
		int(dx) == int(l.width.Get())-1 &&
		int(dy) == int(l.height.Get())-1 {
		return false
	}

	// vertical scroll bar
	if int(dx) == int(l.width.Get()-1) {
		switch {
		case dy == 0:
			l.moveUp(1)
		case dy == types.ACoordY(yy-1):
			l.moveDown(1)
		default:
			newPos := ItemByThumbPosition(int(dy), l.virtualHeight-yy+1, yy)
			if newPos >= 0 {
				l.topLine = newPos
			}
		}

		return true
	}

	// horizontal scrollbar
	switch {
	case dx == 0:
		l.moveLeft()
	case int(dx) == int(l.width.Get())-2:
		l.moveRight()
	default:
		newPos := ItemByThumbPosition(int(dx), l.virtualWidth-int(l.width.Get())+2, int(l.width.Get())-1)
		if newPos >= 0 {
			l.leftShift = newPos
		}
	}

	return true
}

/*
ProcessEvent processes all events come from the control parent. If a control
processes an event it should return true. If the method returns false it means
that the control do not want or cannot process the event and the caller sends
the event to the control parent
*/
func (l *TextView) ProcessEvent(event Event) bool {
	if !l.Active() || !l.Enabled() {
		return false
	}

	switch event.Type {
	case EventKey:
		switch event.Key {
		case term.KeyHome:
			l.home()
			return true
		case term.KeyEnd:
			l.End()
			return true
		case term.KeyArrowUp:
			l.moveUp(1)
			return true
		case term.KeyArrowDown:
			l.moveDown(1)
			return true
		case term.KeyArrowLeft:
			l.moveLeft()
			return true
		case term.KeyArrowRight:
			l.moveRight()
			return true
		case term.KeyPgup:
			l.moveUp(l.outputHeight())
		case term.KeyPgdn:
			l.moveDown(l.outputHeight())
		default:
			return false
		}
	case EventMouse:
		return l.processMouseClick(event)
	}

	return false
}

// own methods

func (l *TextView) calculateVirtualSize() {
	w := l.width.Get() - 1
	l.virtualWidth = int(l.width.Get()) - 1
	l.virtualHeight = 0

	l.lengths = make([]int, len(l.lines))
	for idx, str := range l.lines {
		str = UnColorizeText(str)

		sz := xs.Len(str)
		if l.wordWrap {
			n := sz / int(w)
			r := sz % int(w)
			l.virtualHeight += n
			if r != 0 {
				l.virtualHeight++
			}
		} else {
			l.virtualHeight++
			if sz > l.virtualWidth {
				l.virtualWidth = sz
			}
		}
		l.lengths[idx] = sz
	}
}

// SetText replaces existing content of the control
func (l *TextView) SetText(text []string) {
	l.lines = make([]string, len(text))
	copy(l.lines, text)

	l.applyLimit()
	l.calculateVirtualSize()

	if l.autoscroll {
		l.End()
	}
}

func (l *TextView) posToItemNo(pos int) int {
	id := 0
	for idx, item := range l.lengths {
		if l.virtualWidth >= item {
			pos--
		} else {
			pos -= item / l.virtualWidth
			if item%l.virtualWidth != 0 {
				pos--
			}
		}

		if pos <= 0 {
			id = idx
			break
		}
	}

	return id
}

func (l *TextView) itemNoToPos(id int) int {
	pos := 0
	for i := 0; i < id; i++ {
		if l.virtualWidth >= l.lengths[i] {
			pos++
		} else {
			pos += l.lengths[i] / l.virtualWidth
			if l.lengths[i]%l.virtualWidth != 0 {
				pos++
			}
		}
	}

	return pos
}

// WordWrap returns if the wordwrap is enabled. If the wordwrap
// mode is enabled the control hides horizontal scrollbar and
// draws long lines on a few control lines. There is no
// visual indication if the line is new of it is the portion of
// the previous line yet
func (l *TextView) WordWrap() bool {
	return l.wordWrap
}

func (l *TextView) recalculateTopLine() {
	currLn := l.topLine

	if l.wordWrap {
		l.topLine = l.itemNoToPos(currLn)
	} else {
		l.topLine = l.posToItemNo(currLn)
	}
}

// SetWordWrap enables or disables wordwrap mode
func (l *TextView) SetWordWrap(wrap bool) {
	if wrap != l.wordWrap {
		l.wordWrap = wrap
		l.calculateVirtualSize()
		l.recalculateTopLine()
		l.Draw()
	}
}

// LoadFile loads a text from file and replace the control
// text with the file one.
// Function returns false if loading text from file fails
func (l *TextView) LoadFile(filename string) bool {
	l.lines = make([]string, 0)

	file, err := os.Open(filename)
	if err != nil {
		return false
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimRight(line, " ")
		l.lines = append(l.lines, line)
	}

	l.applyLimit()
	l.calculateVirtualSize()

	if l.autoscroll {
		l.End()
	}

	return true
}

// AutoScroll returns if autoscroll mode is enabled.
// If the autoscroll mode is enabled then the content always
// scrolled to the end after adding a text
func (l *TextView) AutoScroll() bool {
	return l.autoscroll
}

// SetAutoScroll enables and disables autoscroll mode
func (l *TextView) SetAutoScroll(auto bool) {
	l.autoscroll = auto
}

// AddText appends a text to the end of the control content.
// View position may be changed automatically depending on
// value of AutoScroll
func (l *TextView) AddText(text []string) {
	l.lines = append(l.lines, text...)
	l.applyLimit()
	l.calculateVirtualSize()

	if l.autoscroll {
		l.End()
	}
}

// MaxItems returns the maximum number of items that the
// TextView can keep. 0 means unlimited. It makes a TextView
// work like a FIFO queue: the oldest(the first) items are
// deleted if one adds an item to a full TextView
func (l *TextView) MaxItems() int {
	return l.maxLines
}

// SetMaxItems sets the maximum items that TextView keeps
func (l *TextView) SetMaxItems(max int) {
	l.maxLines = max
}

// ItemCount returns the number of items in the TextView
func (l *TextView) ItemCount() int {
	return len(l.lines)
}

func (l *TextView) applyLimit() {
	if l.maxLines == 0 {
		return
	}

	delta := len(l.lines) - l.maxLines
	if delta <= 0 {
		return
	}

	l.lines = l.lines[delta:]
	l.calculateVirtualSize()
	if l.topLine+l.outputHeight() < len(l.lines) {
		l.End()
	}
}
