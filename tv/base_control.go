package tv

import (
	"sync"
	"sync/atomic"

	mTerm "github.com/nsf/termbox-go"
)


var (
	globalRefId int64
)

// возвращает следующий ID виджета
func nextRefId() int64 {
	return atomic.AddInt64(&globalRefId, 1)
}

// NewBasedControl -- возвращает новый TBaseControl
func NewBaseControl() TBaseControl {
	return TBaseControl{
		refID: nextRefId(),
	}
}

// SetClipped -- устанавливает признак обрезки виджета
func (sf *TBaseControl) SetClipped(isClipped bool) {
	sf.isClipped = isClipped
}

func (c *TBaseControl) Clipped() bool {
	return c.isClipped
}

func (c *TBaseControl) SetStyle(style string) {
	c.style = style
}

func (c *TBaseControl) Style() string {
	return c.style
}

func (c *TBaseControl) RefID() int64 {
	return c.refID
}

func (c *TBaseControl) Title() string {
	return c.title
}

func (c *TBaseControl) SetTitle(title string) {
	c.title = title
}

func (c *TBaseControl) Size() (widht int, height int) {
	return c.width, c.height
}

func (c *TBaseControl) SetSize(width, height int) {
	if width < c.minW {
		width = c.minW
	}
	if height < c.minH {
		height = c.minH
	}

	if height != c.height || width != c.width {
		c.height = height
		c.width = width
	}
}

func (c *TBaseControl) Pos() (x int, y int) {
	return c.x, c.y
}

func (c *TBaseControl) SetPos(x, y int) {
	if c.isClipped && c.clipper != nil {
		cx, cy, _, _ := c.Clipper()
		px, py := c.Paddings()

		distX := cx - c.x
		distY := cy - c.y

		c.clipper.x = x + px
		c.clipper.y = y + py

		c.x = (x - distX) + px
		c.y = (y - distY) + py
	} else {
		c.x = x
		c.y = y
	}
}

func (c *TBaseControl) applyConstraints() {
	ww, hh := c.width, c.height
	if ww < c.minW {
		ww = c.minW
	}
	if hh < c.minH {
		hh = c.minH
	}
	if hh != c.height || ww != c.width {
		c.SetSize(ww, hh)
	}
}

func (c *TBaseControl) Constraints() (minw int, minh int) {
	return c.minW, c.minH
}

func (c *TBaseControl) SetConstraints(minw, minh int) {
	c.minW = minw
	c.minH = minh
	c.applyConstraints()
}

func (c *TBaseControl) Active() bool {
	return !c.isInactive
}

func (c *TBaseControl) SetActive(active bool) {
	c.isInactive = !active

	if c.onActive != nil {
		c.onActive(active)
	}
}

func (c *TBaseControl) OnActive(fn func(active bool)) {
	c.onActive = fn
}

func (c *TBaseControl) TabStop() bool {
	return !c.isTabSkip
}

func (c *TBaseControl) SetTabStop(tabstop bool) {
	c.isTabSkip = !tabstop
}

func (c *TBaseControl) Enabled() bool {
	c.block.RLock()
	defer c.block.RUnlock()

	return !c.isDisabled
}

func (c *TBaseControl) SetEnabled(enabled bool) {
	c.block.Lock()
	defer c.block.Unlock()

	c.isDisabled = !enabled
}

func (c *TBaseControl) Visible() bool {
	c.block.RLock()
	defer c.block.RUnlock()

	return !c.isHidden
}

func (c *TBaseControl) SetVisible(visible bool) {
	c.block.Lock()
	defer c.block.Unlock()

	if visible == !c.isHidden {
		return
	}

	c.isHidden = !visible
	if c.parent == nil {
		return
	}

	p := c.Parent()
	for p.Parent() != nil {
		p = p.Parent()
	}

	go func() {
		if FindFirstActiveControl(c) != nil && !c.isInactive {
			PutEvent(Event{Type: EventKey, Key: mTerm.KeyTab})
		}
		PutEvent(Event{Type: EventLayout, Target: p})
	}()
}

func (c *TBaseControl) Parent() types.IWidget {
	return c.parent
}

func (c *TBaseControl) SetParent(parent types.IWidget) {
	if c.parent == nil {
		c.parent = parent
	}
}

func (c *TBaseControl) Modal() bool {
	return c.isModal
}

func (c *TBaseControl) SetModal(modal bool) {
	c.isModal = modal
}

func (c *TBaseControl) Paddings() (px int, py int) {
	return c.padX, c.padY
}

func (c *TBaseControl) SetPaddings(px, py int) {
	if px >= 0 {
		c.padX = px
	}
	if py >= 0 {
		c.padY = py
	}
}

func (c *TBaseControl) Gaps() (dx int, dy int) {
	return c.gapX, c.gapY
}

func (c *TBaseControl) SetGaps(dx, dy int) {
	if dx >= 0 {
		c.gapX = dx
	}
	if dy >= 0 {
		c.gapY = dy
	}
}

func (c *TBaseControl) Pack() PackType {
	return c.pack
}

func (c *TBaseControl) SetPack(pack PackType) {
	c.pack = pack
}

func (c *TBaseControl) Scale() int {
	return c.scale
}

func (c *TBaseControl) SetScale(scale int) {
	if scale >= 0 {
		c.scale = scale
	}
}

func (c *TBaseControl) Align() Align {
	return c.align
}

func (c *TBaseControl) SetAlign(align Align) {
	c.align = align
}

func (c *TBaseControl) TextColor() mTerm.Attribute {
	return c.fg
}

func (c *TBaseControl) SetTextColor(clr mTerm.Attribute) {
	c.fg = clr
}

func (c *TBaseControl) BackColor() mTerm.Attribute {
	return c.bg
}

func (c *TBaseControl) SetBackColor(clr mTerm.Attribute) {
	c.bg = clr
}

func (c *TBaseControl) childCount() int {
	cnt := 0
	for _, child := range c.children {
		if child.Visible() {
			cnt++
		}
	}

	return cnt
}

func (c *TBaseControl) ResizeChildren() {
	children := c.childCount()
	if children == 0 {
		return
	}

	fullWidth := c.width - 2*c.padX
	fullHeight := c.height - 2*c.padY
	if c.pack == Horizontal {
		fullWidth -= (children - 1) * c.gapX
	} else {
		fullHeight -= (children - 1) * c.gapY
	}

	totalSc := c.ChildrenScale()
	minWidth := 0
	minHeight := 0
	for _, child := range c.children {
		if !child.Visible() {
			continue
		}

		cw, ch := child.MinimalSize()
		if c.pack == Horizontal {
			minWidth += cw
		} else {
			minHeight += ch
		}
	}

	aStep := 0
	diff := fullWidth - minWidth
	if c.pack == Vertical {
		diff = fullHeight - minHeight
	}
	if totalSc > 0 {
		aStep = int(float32(diff) / float32(totalSc))
	}

	for _, ctrl := range c.children {
		if !ctrl.Visible() {
			continue
		}

		tw, th := ctrl.MinimalSize()
		sc := ctrl.Scale()
		d := ctrl.Scale() * aStep
		if c.pack == Horizontal {
			if sc != 0 {
				if sc == totalSc {
					tw += diff
					d = diff
				} else {
					tw += d
				}
			}
			th = fullHeight
		} else {
			if sc != 0 {
				if sc == totalSc {
					th += diff
					d = diff
				} else {
					th += d
				}
			}
			tw = fullWidth
		}
		diff -= d
		totalSc -= sc

		ctrl.SetSize(tw, th)
		ctrl.ResizeChildren()
	}
}

func (c *TBaseControl) AddChild(control types.IWidget) {
	if c.children == nil {
		c.children = make([]types.IWidget, 1)
		c.children[0] = control
	} else {
		if c.ChildExists(control) {
			panic("Double adding a child")
		}

		c.children = append(c.children, control)
	}

	var ctrl types.IWidget
	var mainCtrl types.IWidget
	ctrl = c
	for ctrl != nil {
		ww, hh := ctrl.MinimalSize()
		cw, ch := ctrl.Size()
		if ww > cw || hh > ch {
			if ww > cw {
				cw = ww
			}
			if hh > ch {
				ch = hh
			}
			ctrl.SetConstraints(cw, ch)
		}

		if ctrl.Parent() == nil {
			mainCtrl = ctrl
		}
		ctrl = ctrl.Parent()
	}

	if mainCtrl != nil {
		mainCtrl.ResizeChildren()
		mainCtrl.PlaceChildren()
	}

	if c.isClipped && c.clipper == nil {
		c.setClipper()
	}
}

func (c *TBaseControl) Children() []types.IWidget {
	child := make([]types.IWidget, len(c.children))
	copy(child, c.children)
	return child
}

func (c *TBaseControl) ChildExists(control types.IWidget) bool {
	if len(c.children) == 0 {
		return false
	}

	for _, ctrl := range c.children {
		if ctrl == control {
			return true
		}
	}

	return false
}

func (c *TBaseControl) ChildrenScale() int {
	if c.childCount() == 0 {
		return c.scale
	}

	total := 0
	for _, ctrl := range c.children {
		if ctrl.Visible() {
			total += ctrl.Scale()
		}
	}

	return total
}

func (c *TBaseControl) MinimalSize() (w int, h int) {
	children := c.childCount()
	if children == 0 {
		return c.minW, c.minH
	}

	totalX := 2 * c.padX
	totalY := 2 * c.padY

	if c.pack == Vertical {
		totalY += (children - 1) * c.gapY
	} else {
		totalX += (children - 1) * c.gapX
	}

	for _, ctrl := range c.children {
		if ctrl.Clipped() {
			continue
		}

		if !ctrl.Visible() {
			continue
		}
		ww, hh := ctrl.MinimalSize()
		if c.pack == Vertical {
			totalY += hh
			if ww+2*c.padX > totalX {
				totalX = ww + 2*c.padX
			}
		} else {
			totalX += ww
			if hh+2*c.padY > totalY {
				totalY = hh + 2*c.padY
			}
		}
	}

	if totalX < c.minW {
		totalX = c.minW
	}
	if totalY < c.minH {
		totalY = c.minH
	}

	return totalX, totalY
}

func (c *TBaseControl) Draw() {
	panic("BaseControl Draw Called")
}

func (c *TBaseControl) DrawChildren() {
	if c.isHidden {
		return
	}

	PushClip()
	defer PopClip()

	cp := ClippedParent(c)
	var cTarget types.IWidget

	cTarget = c
	if cp != nil {
		cTarget = cp
	}

	x, y, w, h := cTarget.Clipper()
	SetClipRect(x, y, w, h)

	for _, child := range c.children {
		child.Draw()
	}
}

func (c *TBaseControl) Clipper() (int, int, int, int) {
	clipped := ClippedParent(c)

	if clipped == nil || (c.isClipped && c.clipper != nil) {
		return c.clipper.x, c.clipper.y, c.clipper.w, c.clipper.h
	}

	return CalcClipper(c)
}

func (c *TBaseControl) setClipper() {
	x, y, w, h := CalcClipper(c)
	c.clipper = &rect{x: x, y: y, w: w, h: h}
}

func (c *TBaseControl) HitTest(x, y int) HitResult {
	if x > c.x && x < c.x+c.width-1 &&
		y > c.y && y < c.y+c.height-1 {
		return HitInside
	}

	if (x == c.x || x == c.x+c.width-1) &&
		y >= c.y && y < c.y+c.height {
		return HitBorder
	}

	if (y == c.y || y == c.y+c.height-1) &&
		x >= c.x && x < c.x+c.width {
		return HitBorder
	}

	return HitOutside
}

func (c *TBaseControl) ProcessEvent(ev Event) bool {
	return SendEventToChild(c, ev)
}

func (c *TBaseControl) PlaceChildren() {
	children := c.childCount()
	if c.children == nil || children == 0 {
		return
	}

	xx, yy := c.x+c.padX, c.y+c.padY
	for _, ctrl := range c.children {
		if !ctrl.Visible() {
			continue
		}

		ctrl.SetPos(xx, yy)
		ww, hh := ctrl.Size()
		if c.pack == Vertical {
			yy += c.gapY + hh
		} else {
			xx += c.gapX + ww
		}

		ctrl.PlaceChildren()
	}
}

// ActiveColors return the attributes for the controls when it
// is active: text and background colors
func (c *TBaseControl) ActiveColors() (mTerm.Attribute, mTerm.Attribute) {
	return c.fgActive, c.bgActive
}

// SetActiveTextColor changes text color of the active control
func (c *TBaseControl) SetActiveTextColor(clr mTerm.Attribute) {
	c.fgActive = clr
}

// SetActiveBackColor changes background color of the active control
func (c *TBaseControl) SetActiveBackColor(clr mTerm.Attribute) {
	c.bgActive = clr
}

func (c *TBaseControl) removeChild(control types.IWidget) {
	children := []types.IWidget{}

	for _, child := range c.children {
		if child.RefID() == control.RefID() {
			continue
		}

		children = append(children, child)
	}
	c.children = nil

	for _, child := range children {
		c.AddChild(child)
	}
}

// Destroy removes an object from its parental chain
func (c *TBaseControl) Destroy() {
	c.parent.removeChild(c)
	c.parent.SetConstraints(0, 0)
}
