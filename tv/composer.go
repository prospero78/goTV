package tv

import (
	"sync"

	term "github.com/nsf/termbox-go"

	"github.com/prospero78/goTV/tv/types"
)

// Composer is a service object that manages Views and console, processes
// events, and provides service methods. One application must have only
// one object of this type
type Composer struct {
	// list of visible Views
	windows      []IControl
	windowBorder BorderStyle
	consumer     IControl
	// last pressed key - to make repeatable actions simpler, e.g, at first
	// one presses Ctrl+S and then just repeatedly presses arrow lest to
	// resize Window
	lastKey term.Key
	// coordinates when the mouse button was down, e.g to detect
	// mouse click
	mdownX types.ACoordX
	mdownY types.ACoordY
	// last processed coordinates: e.g, for mouse move
	lastX types.ACoordX
	lastY types.ACoordY
	// Type of dragging
	dragType DragType
	// For safe Window manipulations
	mtx sync.RWMutex
}

var (
	comp *Composer
)

func initComposer() {
	comp = new(Composer)
	comp.windows = make([]IControl, 0)
	comp.windowBorder = BorderAuto
	comp.consumer = nil
	comp.lastKey = term.KeyEsc
}

// WindowManager returns main Window manager (that is Composer). Use it at
// your own risk because it provides an access to some low level Window
// manipulations.
// Note: Now it is not thread safe to call Composer methods from a few threads.
func WindowManager() *Composer {
	return comp
}

// GrabEvents makes control c as the exclusive event reciever. After calling
// this function the control will recieve all mouse and keyboard events even
// if it is not active or mouse is outside it. Useful to implement dragging
// or alike stuff
func GrabEvents(c IControl) {
	comp.consumer = c
}

// ReleaseEvents stops a control being exclusive evetn reciever and backs all
// to normal event processing
func ReleaseEvents() {
	comp.consumer = nil
}

func termboxEventToLocal(ev term.Event) Event {
	e := Event{Type: EventType(ev.Type), Ch: ev.Ch,
		Key: ev.Key, Err: ev.Err, X: types.ACoordX(ev.MouseX), Y: types.ACoordY(ev.MouseY),
		Mod: ev.Mod, Width: ev.Width, Height: ev.Height}
	return e
}

// Repaints everything on the screen
func RefreshScreen() {
	comp.BeginUpdate()
	_ = term.Clear(ColorWhite, ColorBlack)
	comp.EndUpdate()

	windows := comp.getWindowList()
	for _, wnd := range windows {
		v := wnd.(*TWindow)
		if v.Visible() {
			wnd.Draw()

			WindowManager().BeginUpdate()
			PushAttributes()
			term.Flush()
			PopAttributes()
			WindowManager().EndUpdate()

		}
	}

	comp.BeginUpdate()
	term.Flush()
	comp.EndUpdate()
}

// AddWindow constucts a new Window, adds it to the composer automatically,
// and makes it active
// posX and posY are top left coordinates of the Window
// width and height are Window size
// title is a Window title
func AddWindow(posX types.ACoordX, posY types.ACoordY,
	width, height int,
	title string,
	autoWidth types.AAutoWidth,
	autoHeight types.AAutoHeight) *TWindow {
	window := NewWindow(posX, posY, width, height, title, autoWidth, autoHeight)
	window.SetBorder(comp.windowBorder)

	comp.BeginUpdate()
	comp.windows = append(comp.windows, window)
	comp.EndUpdate()
	window.Draw()
	term.Flush()

	comp.activateWindow(window)

	RefreshScreen()

	return window
}

// Border returns the default window border
func (c *Composer) BorderStyle() BorderStyle {
	return c.windowBorder
}

// SetBorder changes the default window border
func (c *Composer) SetBorder(border BorderStyle) {
	c.windowBorder = border
}

// BeginUpdate locks any screen update until EndUpdate is called.
// Useful only in multithreading application if you create a new Window in
// some thread that is not main one (e.g, create new Window inside
// OnSelectItem handler of ListBox)
// Note: Do not lock for a long time because while the lock is on the screen is
// not updated
func (c *Composer) BeginUpdate() {
	c.mtx.Lock()
}

// EndUpdate unlocks the screen for any manipulations.
// Useful only in multithreading application if you create a new Window in
// some thread that is not main one (e.g, create new Window inside
// OnSelectItem handler of ListBox)
func (c *Composer) EndUpdate() {
	c.mtx.Unlock()
}

func (c *Composer) getWindowList() []IControl {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	arr_copy := make([]IControl, len(c.windows))
	copy(arr_copy, c.windows)
	return arr_copy
}

func (c *Composer) checkWindowUnderMouse(screenX types.ACoordX, screenY types.ACoordY) (IControl, HitResult) {
	windows := c.getWindowList()
	if len(windows) == 0 {
		return nil, HitOutside
	}

	for i := len(windows) - 1; i >= 0; i-- {
		window := windows[i]
		hit := window.HitTest(screenX, screenY)
		if hit != HitOutside {
			return window, hit
		}
	}

	return nil, HitOutside
}

func (c *Composer) activateWindow(window IControl) bool {
	windows := c.getWindowList()
	if c.topWindow() == window {
		for _, v := range windows {
			v.SetActive(false)
		}
		window.SetActive(true)
		return true
	}

	var wList []IControl
	found := false

	for _, v := range windows {
		if v != window {
			v.SetActive(false)
			wList = append(wList, v)
		} else {
			found = true
		}
	}

	if !found {
		return false
	}

	window.SetActive(true)
	c.BeginUpdate()
	defer c.EndUpdate()
	c.windows = append(wList, window)
	return true
}

func (c *Composer) moveActiveWindowToBottom() bool {
	windows := c.getWindowList()
	if len(windows) < 2 {
		return false
	}

	if c.topWindow().Modal() {
		return false
	}

	anyVisible := false
	for _, w := range windows {
		v := w.(*TWindow)
		if v.Visible() {
			anyVisible = true
			break
		}
	}
	if !anyVisible {
		return false
	}

	event := Event{Type: EventActivate, X: 0} // send deactivated
	c.sendEventToActiveWindow(event)

	for {
		last := c.topWindow()
		c.BeginUpdate()
		for i := len(c.windows) - 1; i > 0; i-- {
			c.windows[i] = c.windows[i-1]
		}
		c.windows[0] = last
		c.EndUpdate()

		v := c.topWindow().(*TWindow)
		if v.Visible() {
			if !c.activateWindow(c.topWindow()) {
				return false
			}

			break
		}
	}

	event = Event{Type: EventActivate, X: 1} // send 'activated'
	c.sendEventToActiveWindow(event)
	RefreshScreen()

	return true
}

func (c *Composer) sendEventToActiveWindow(ev Event) bool {
	view := c.topWindow()
	if view != nil {
		return view.ProcessEvent(ev)
	}

	return false
}

func (c *Composer) topWindow() IControl {
	windows := c.getWindowList()

	if len(windows) == 0 {
		return nil
	}

	return windows[len(windows)-1]
}

func (c *Composer) resizeTopWindow(ev Event) bool {
	view := c.topWindow()
	if view == nil {
		return false
	}

	topwindow, ok := view.(*TWindow)
	if ok && !topwindow.Sizable() {
		return false
	}

	w, h := view.Size()
	w1, h1 := w, h
	minW, minH := view.Constraints()
	switch {
	case ev.Key == term.KeyArrowUp && minH < h:
		h--
	case ev.Key == term.KeyArrowLeft && minW < w:
		w--
	case ev.Key == term.KeyArrowDown:
		h++
	case ev.Key == term.KeyArrowRight:
		w++
	}

	if w1 != w || h1 != h {
		view.SetSize(w, h)
		event := Event{Type: EventResize, X: types.ACoordX(w), Y: types.ACoordY(h)}
		c.sendEventToActiveWindow(event)
		RefreshScreen()
	}

	return true
}

func (c *Composer) moveTopWindow(ev Event) bool {
	view := c.topWindow()
	if view != nil {
		topwindow, ok := view.(*TWindow)
		if ok && !topwindow.Movable() {
			return false
		}

		x, y := view.Pos().Get()
		w, h := view.Size()
		x1, y1 := x, y
		cx, cy := term.Size()
		switch {
		case ev.Key == term.KeyArrowUp && y > 0:
			y--
		case ev.Key == term.KeyArrowDown && int(y)+h < cy:
			y++
		case ev.Key == term.KeyArrowLeft && x > 0:
			x--
		case ev.Key == term.KeyArrowRight && x+types.ACoordX(w) < types.ACoordX(cx):
			x++
		}

		if x1 != x || y1 != y {
			view.SetPos(x, y)
			event := Event{Type: EventMove, X: x, Y: y}
			c.sendEventToActiveWindow(event)
			RefreshScreen()
		}
		return true
	}

	return false
}

func (c *Composer) closeTopWindow() {
	if len(c.windows) > 1 {
		view := c.topWindow()
		event := Event{Type: EventClose, X: 1}

		if c.sendEventToActiveWindow(event) {
			c.DestroyWindow(view)
			activate := c.topWindow()
			c.activateWindow(activate)
			event = Event{Type: EventActivate, X: 1} // send 'activated'
			c.sendEventToActiveWindow(event)
		}

		RefreshScreen()
	} else {
		go Stop()
	}
}

func (c *Composer) processWindowDrag(ev Event) {
	if ev.Mod != term.ModMotion || c.dragType == DragNone {
		return
	}
	dx := ev.X - c.lastX
	dy := ev.Y - c.lastY
	if dx == 0 && dy == 0 {
		return
	}

	w := c.topWindow()
	newX, newY := w.Pos().Get()
	newW, newH := w.Size()
	cw, ch := ScreenSize()

	switch c.dragType {
	case DragMove:
		newX += dx
		newY += dy
		if newX >= 0 && newY >= 0 && newX+types.ACoordX(newW) < types.ACoordX(cw) && int(newY)+newH < ch {
			c.lastX = ev.X
			c.lastY = ev.Y

			w.SetPos(newX, newY)
			event := Event{Type: EventMove, X: newX, Y: newY}
			c.sendEventToActiveWindow(event)
			RefreshScreen()
		}
	case DragResizeLeft:
		newX += dx
		newW -= int(dx)
		if newX >= 0 && newY >= 0 && newX+types.ACoordX(newW) < types.ACoordX(cw) && int(newY)+newH < ch {
			c.lastX = ev.X
			c.lastY = ev.Y

			w.SetPos(newX, newY)
			w.SetSize(newW, newH)
			event := Event{Type: EventMove, X: newX, Y: newY}
			c.sendEventToActiveWindow(event)
			event.Type = EventResize
			c.sendEventToActiveWindow(event)
			RefreshScreen()
		}
	case DragResizeRight:
		newW += int(dx)
		if newX >= 0 && newY >= 0 && int(newX)+newW < cw && int(newY)+newH < ch {
			c.lastX = ev.X
			c.lastY = ev.Y

			w.SetSize(newW, newH)
			event := Event{Type: EventResize}
			c.sendEventToActiveWindow(event)
			RefreshScreen()
		}
	case DragResizeBottom:
		newH += int(dy)
		if newX >= 0 && newY >= 0 && int(newX)+newW < cw && int(newY)+newH < ch {
			c.lastX = ev.X
			c.lastY = ev.Y

			w.SetSize(newW, newH)
			event := Event{Type: EventResize}
			c.sendEventToActiveWindow(event)
			RefreshScreen()
		}
	case DragResizeTopLeft:
		newX += dx
		newW -= int(dx)
		newY += dy
		newH -= int(dy)
		if newX >= 0 && newY >= 0 && int(newX)+newW < cw && int(newY)+newH < ch {
			c.lastX = ev.X
			c.lastY = ev.Y

			w.SetPos(newX, newY)
			w.SetSize(newW, newH)
			event := Event{Type: EventMove, X: newX, Y: newY}
			c.sendEventToActiveWindow(event)
			event.Type = EventResize
			c.sendEventToActiveWindow(event)
			RefreshScreen()
		}
	case DragResizeBottomLeft:
		newX += dx
		newW -= int(dx)
		newH += int(dy)
		if newX >= 0 && newY >= 0 && int(newX)+newW < cw && int(newY)+newH < ch {
			c.lastX = ev.X
			c.lastY = ev.Y

			w.SetPos(newX, newY)
			w.SetSize(newW, newH)
			event := Event{Type: EventMove, X: newX, Y: newY}
			c.sendEventToActiveWindow(event)
			event.Type = EventResize
			c.sendEventToActiveWindow(event)
			RefreshScreen()
		}
	case DragResizeBottomRight:
		newW += int(dx)
		newH += int(dy)
		if newX >= 0 && newY >= 0 && int(newX)+newW < cw && int(newY)+newH < ch {
			c.lastX = ev.X
			c.lastY = ev.Y

			w.SetSize(newW, newH)
			event := Event{Type: EventResize}
			c.sendEventToActiveWindow(event)
			RefreshScreen()
		}
	case DragResizeTopRight:
		newY += dy
		newW += int(dx)
		newH -= int(dy)
		if newX >= 0 && newY >= 0 && int(newX)+newW < cw && int(newY)+newH < ch {
			c.lastX = ev.X
			c.lastY = ev.Y

			w.SetPos(newX, newY)
			w.SetSize(newW, newH)
			event := Event{Type: EventMove, X: newX, Y: newY}
			c.sendEventToActiveWindow(event)
			event.Type = EventResize
			c.sendEventToActiveWindow(event)
			RefreshScreen()
		}
	}
}

func (c *Composer) processMouse(ev Event) {
	if c.consumer != nil {
		tmp := c.consumer
		tmp.ProcessEvent(ev)
		tmp.Draw()
		term.Flush()
		return
	}

	view, hit := c.checkWindowUnderMouse(ev.X, ev.Y)
	if c.dragType != DragNone {
		view = c.topWindow()
	}

	if c.topWindow() == view {
		if ev.Key == term.MouseRelease && c.dragType != DragNone {
			c.dragType = DragNone
			return
		}

		if ev.Mod == term.ModMotion && c.dragType != DragNone {
			c.processWindowDrag(ev)
			return
		}

		if hit != HitInside && ev.Key == term.MouseLeft {
			if hit != HitButtonClose && hit != HitButtonBottom && hit != HitButtonMaximize {
				c.lastX = ev.X
				c.lastY = ev.Y
				c.mdownX = ev.X
				c.mdownY = ev.Y
			}
			switch hit {
			case HitButtonClose:
				c.closeTopWindow()
			case HitButtonBottom:
				c.moveActiveWindowToBottom()
			case HitButtonMaximize:
				v := c.topWindow().(*TWindow)
				maximized := v.Maximized()
				v.SetMaximized(!maximized)
			case HitTop:
				c.dragType = DragMove
			case HitBottom:
				c.dragType = DragResizeBottom
			case HitLeft:
				c.dragType = DragResizeLeft
			case HitRight:
				c.dragType = DragResizeRight
			case HitTopLeft:
				c.dragType = DragResizeTopLeft
			case HitTopRight:
				c.dragType = DragResizeTopRight
			case HitBottomRight:
				c.dragType = DragResizeBottomRight
			case HitBottomLeft:
				c.dragType = DragResizeBottomLeft
			}

			return
		}
	} else if !c.topWindow().Modal() {
		c.activateWindow(view)
		return
	}
	switch {
	case ev.Key == term.MouseLeft:
		c.lastX = ev.X
		c.lastY = ev.Y
		c.mdownX = ev.X
		c.mdownY = ev.Y
		c.sendEventToActiveWindow(ev)
	case ev.Key == term.MouseRelease:
		c.sendEventToActiveWindow(ev)
		if c.lastX != ev.X && c.lastY != ev.Y {
			return
		}

		ev.Type = EventClick
		c.sendEventToActiveWindow(ev)
	default:
		c.sendEventToActiveWindow(ev)
	}
}

// Stop sends termination event to Composer. Composer should stop
// console management and quit application
func Stop() {
	ev := Event{Type: EventQuit}
	go PutEvent(ev)
}

// DestroyWindow removes the Window from the list of managed Windows
func (c *Composer) DestroyWindow(view IControl) {
	ev := Event{Type: EventClose}
	c.sendEventToActiveWindow(ev)

	windows := c.getWindowList()
	var newOrder []IControl
	for i := 0; i < len(windows); i++ {
		if windows[i] != view {
			newOrder = append(newOrder, windows[i])
		}
	}

	if len(newOrder) == 0 {
		go Stop()
		return
	}

	c.BeginUpdate()
	c.windows = newOrder
	c.EndUpdate()
	c.activateWindow(c.topWindow())
}

// IsDeadKey returns true if the pressed key is the first key in
// the key sequence understood by composer. Dead key is never sent to
// any control
func IsDeadKey(key term.Key) bool {
	if key == term.KeyCtrlS || key == term.KeyCtrlP ||
		key == term.KeyCtrlW || key == term.KeyCtrlQ {
		return true
	}

	return false
}

func (c *Composer) processKey(ev Event) {
	if ev.Key == term.KeyEsc {
		if IsDeadKey(c.lastKey) {
			c.lastKey = term.KeyEsc
			return
		}
	}

	if IsDeadKey(ev.Key) && !IsDeadKey(c.lastKey) {
		c.lastKey = ev.Key
		return
	}

	if !IsDeadKey(ev.Key) {
		if c.consumer != nil {
			tmp := c.consumer
			tmp.ProcessEvent(ev)
			tmp.Draw()
			term.Flush()
		} else {
			c.sendEventToActiveWindow(ev)
			c.topWindow().Draw()
			term.Flush()
		}
	}

	newKey := term.KeyEsc
	switch c.lastKey {
	case term.KeyCtrlQ:
		switch ev.Key {
		case term.KeyCtrlQ:
			Stop()
		default:
			newKey = ev.Key
		}
	case term.KeyCtrlS:
		switch ev.Key {
		case term.KeyArrowUp, term.KeyArrowDown, term.KeyArrowLeft, term.KeyArrowRight:
			c.resizeTopWindow(ev)
		default:
			newKey = ev.Key
		}
	case term.KeyCtrlP:
		switch ev.Key {
		case term.KeyArrowUp, term.KeyArrowDown, term.KeyArrowLeft, term.KeyArrowRight:
			c.moveTopWindow(ev)
		default:
			newKey = ev.Key
		}
	case term.KeyCtrlW:
		switch ev.Key {
		case term.KeyCtrlH:
			c.moveActiveWindowToBottom()
		case term.KeyCtrlM:
			w := c.topWindow().(*TWindow)
			if w.Sizable() && (w.TitleButtons()&ButtonMaximize == ButtonMaximize) {
				maxxed := w.Maximized()
				w.SetMaximized(!maxxed)
				RefreshScreen()
			}
		case term.KeyCtrlC:
			c.closeTopWindow()
		default:
			newKey = ev.Key
		}
	}

	if newKey != term.KeyEsc {
		event := Event{Key: c.lastKey, Type: EventKey}
		c.sendEventToActiveWindow(event)
		event.Key = newKey
		c.sendEventToActiveWindow(event)
		c.lastKey = term.KeyEsc
	}
}

func ProcessEvent(ev Event) {
	switch ev.Type {
	case EventCloseWindow:
		comp.closeTopWindow()
	case EventRedraw:
		RefreshScreen()
	case EventResize:
		SetScreenSize(ev.Width, ev.Height)
		for _, c := range comp.windows {
			wnd := c.(*TWindow)
			if wnd.Maximized() {
				wnd.SetSize(ev.Width, ev.Height)
				wnd.ResizeChildren()
				wnd.PlaceChildren()
				RefreshScreen()
			}

			if wnd.onScreenResize != nil {
				wnd.onScreenResize(ev)
			}

		}
	case EventKey:
		comp.processKey(ev)
	case EventMouse:
		comp.processMouse(ev)
	case EventLayout:
		for _, c := range comp.windows {
			if c == ev.Target {
				c.ResizeChildren()
				c.PlaceChildren()
				break
			}
		}
	}
}
