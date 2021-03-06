package tv

import (
	"strconv"
	"strings"

	xs "github.com/huandu/xstrings"
	term "github.com/nsf/termbox-go"

	"github.com/prospero78/goTV/tv/autoheight"
	"github.com/prospero78/goTV/tv/autowidth"
	"github.com/prospero78/goTV/tv/types"
)

/*
ProgressBar control visualizes the progression of extended operation.

The control has two sets of colors(almost all other controls have only
one set: foreground and background colors): for filled part and for
empty one. By default colors are the same.
*/
type ProgressBar struct {
	TBaseControl
	direction        Direction
	min, max         int
	value            int
	emptyFg, emptyBg term.Attribute
	titleFg          term.Attribute
	autoWidth        types.IAutoWidth
	autoHeight       types.IAutoHeight
}

/*
CreateProgressBar creates a new ProgressBar.
parent - is container that keeps the control.
width and heigth - are minimal size of the control.
scale - the way of scaling the control when the parent is resized. Use DoNotScale constant if the
control should keep its original size.
*/
func CreateProgressBar(parent IControl, width, height int, scale int) *ProgressBar {
	b := &ProgressBar{
		TBaseControl: NewBaseControl(),
		autoWidth:    autowidth.New(),
		autoHeight:   autoheight.New(),
	}

	if height == 0 {
		height = 1
		b.autoHeight.Set()
	}
	if width == 0 {
		width = 10
		b.autoWidth.Set()
	}

	b.SetSize(width, height)
	b.SetConstraints(width, height)
	b.SetTabStop(false)
	b.SetScale(scale)
	b.min = 0
	b.max = 10
	b.direction = Horizontal
	b.parent = parent
	b.align = AlignCenter

	if parent != nil {
		parent.AddChild(b)
	}

	return b
}

// Draw repaints the control on its View surface.
// Horizontal ProgressBar supports custom title over the bar.
// One can set title using method SetTitle. There are a few
// predefined variables that can be used inside title to
// show value or total progress. Variable must be enclosed
// with double curly brackets. Available variables:
// percent - the current percentage rounded to closest integer
// value - raw ProgressBar value
// min - lower ProgressBar limit
// max - upper ProgressBar limit
// Examples:
//      pb.SetTitle("{{value}} of {{max}}")
//      pb.SetTitle("{{percent}}%")
func (b *ProgressBar) Draw() {
	if b.hidden {
		return
	}

	b.mtx.RLock()
	defer b.mtx.RUnlock()
	if b.max <= b.min {
		return
	}

	PushAttributes()
	defer PopAttributes()

	fgOff, fgOn := RealColor(b.fg, b.Style(), ColorProgressText), RealColor(b.fgActive, b.Style(), ColorProgressActiveText)
	bgOff, bgOn := RealColor(b.bg, b.Style(), ColorProgressBack), RealColor(b.bgActive, b.Style(), ColorProgressActiveBack)

	parts := []rune(SysObject(ObjProgressBar))
	cFilled, cEmpty := parts[0], parts[1]

	prc := 0
	if b.value >= b.max {
		prc = 100
	} else if b.value < b.max && b.value > b.min {
		prc = (100 * (b.value - b.min)) / (b.max - b.min)
	}

	var title string
	if b.direction == Horizontal && b.Title() != "" {
		title = b.Title()
		title = strings.ReplaceAll(title, "{{percent}}", strconv.Itoa(prc))
		title = strings.ReplaceAll(title, "{{value}}", strconv.Itoa(b.value))
		title = strings.ReplaceAll(title, "{{min}}", strconv.Itoa(b.min))
		title = strings.ReplaceAll(title, "{{max}}", strconv.Itoa(b.max))
	}

	x, y := b.pos.Get()
	w, h := b.Size()

	if b.direction == Horizontal {
		filled := prc * w / 100
		sFilled := strings.Repeat(string(cFilled), filled)
		sEmpty := strings.Repeat(string(cEmpty), w-filled)

		for yy := y; yy < y+types.ACoordY(h); yy++ {
			SetTextColor(fgOn)
			SetBackColor(bgOn)
			DrawRawText(x, yy, sFilled)
			SetTextColor(fgOff)
			SetBackColor(bgOff)
			DrawRawText(x+types.ACoordX(filled), yy, sEmpty)
		}

		if title != "" {
			shift, str := AlignText(title, w, b.align)
			titleClr := RealColor(b.titleFg, b.Style(), ColorProgressTitleText)
			var sOn, sOff string
			switch {
			case filled == 0 || shift >= filled:
				sOff = str
			case w == filled || shift+xs.Len(str) < filled:
				sOn = str
			default:
				r := filled - shift
				sOn = xs.Slice(str, 0, r)
				sOff = xs.Slice(str, r, -1)
			}

			SetTextColor(titleClr)
			if sOn != "" {
				SetBackColor(bgOn)
				DrawRawText(x+types.ACoordX(shift), y, sOn)
			}
			if sOff != "" {
				SetBackColor(bgOff)
				DrawRawText(x+types.ACoordX(shift+xs.Len(sOn)), y, sOff)
			}
		}
	} else {
		filled := prc * h / 100
		sFilled := strings.Repeat(string(cFilled), w)
		sEmpty := strings.Repeat(string(cEmpty), w)
		for yy := y; yy < y+types.ACoordY(h-filled); yy++ {
			SetTextColor(fgOff)
			SetBackColor(bgOff)
			DrawRawText(x, yy, sEmpty)
		}
		for yy := y + types.ACoordY(h-filled); yy < y+types.ACoordY(h); yy++ {
			SetTextColor(fgOff)
			SetBackColor(bgOff)
			DrawRawText(x, yy, sFilled)
		}
	}
}

//----------------- own methods -------------------------

// SetValue sets new progress value. If value exceeds ProgressBar
// limits then the limit value is used
func (b *ProgressBar) SetValue(pos int) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	switch {
	case pos < b.min:
		b.value = b.min
	case pos > b.max:
		b.value = b.max
	default:
		b.value = pos
	}
}

// Value returns the current ProgressBar value
func (b *ProgressBar) Value() int {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	return b.value
}

// Limits returns current minimal and maximal values of ProgressBar
func (b *ProgressBar) Limits() (int, int) {
	return b.min, b.max
}

// SetLimits set new ProgressBar limits. The current value
// is adjusted if it exceeds new limits
func (b *ProgressBar) SetLimits(min, max int) {
	b.min = min
	b.max = max

	if b.value < b.min {
		b.value = min
	}
	if b.value > b.max {
		b.value = max
	}
}

// Step increases ProgressBar value by 1 if the value is less
// than ProgressBar high limit
func (b *ProgressBar) Step() int {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.value++

	if b.value > b.max {
		b.value = b.max
	}

	return b.value
}

// SecondaryColors returns text and background colors for empty
// part of the ProgressBar
func (b *ProgressBar) SecondaryColors() (term.Attribute, term.Attribute) {
	return b.emptyFg, b.emptyBg
}

// SetSecondaryColors sets new text and background colors for
// empty part of the ProgressBar
func (b *ProgressBar) SetSecondaryColors(fg, bg term.Attribute) {
	b.emptyFg, b.emptyBg = fg, bg
}

// TitleColor returns text color of ProgressBar's title. Title
// background color always equals background color of the
// part of the ProgressBar on which it is displayed. In other
// words, background color of title is transparent
func (b *ProgressBar) TitleColor() term.Attribute {
	return b.titleFg
}

// SetTitleColor sets text color of ProgressBar's title
func (b *ProgressBar) SetTitleColor(clr term.Attribute) {
	b.titleFg = clr
}
