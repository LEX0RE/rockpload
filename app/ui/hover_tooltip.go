package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

//go:embed icons/controller.svg
var controllerSVG []byte

var controllerIcon = theme.NewThemedResource(fyne.NewStaticResource("controller.svg", controllerSVG))

type hoverTooltip struct {
	layer      *fyne.Container
	card       *fyne.Container
	background *canvas.Rectangle
	label      *widget.Label
}

func newHoverTooltip() *hoverTooltip {
	t := &hoverTooltip{
		background: canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)),
		label:      widget.NewLabel(""),
	}

	t.label.Wrapping = fyne.TextWrapWord
	t.background.StrokeWidth = 1
	t.background.CornerRadius = theme.InputRadiusSize()

	t.card = container.NewStack(t.background, container.NewPadded(t.label))
	t.card.Hide()
	t.layer = container.NewWithoutLayout(t.card)

	return t
}

func (t *hoverTooltip) Over(content fyne.CanvasObject) fyne.CanvasObject {
	return container.NewStack(content, t.layer)
}

func (t *hoverTooltip) Show(anchor fyne.CanvasObject, text string) {
	driver := fyne.CurrentApp().Driver()
	anchorPos := driver.AbsolutePositionForObject(anchor).Subtract(driver.AbsolutePositionForObject(t.layer))
	area := t.layer.Size()

	width := min(float32(tooltipWidth+tooltipPadding), area.Width)
	t.label.SetText(text)
	t.label.Resize(fyne.NewSize(width-2*theme.Padding(), 0))
	size := fyne.NewSize(width, t.label.MinSize().Height+2*theme.Padding())

	x := min(max(anchorPos.X+anchor.Size().Width-size.Width, 0), area.Width-size.Width)
	y := anchorPos.Y + anchor.Size().Height
	if y+size.Height > area.Height {
		y = max(anchorPos.Y-size.Height, 0)
	}

	t.background.FillColor = theme.Color(theme.ColorNameOverlayBackground)
	t.background.StrokeColor = theme.Color(theme.ColorNameSeparator)

	t.card.Resize(size)
	t.card.Move(fyne.NewPos(x, y))
	t.card.Show()
	t.card.Refresh()
}

func (t *hoverTooltip) Hide() {
	t.card.Hide()
}

type hoverIcon struct {
	widget.Icon

	tooltip *hoverTooltip
	Text    string
}

var _ desktop.Hoverable = (*hoverIcon)(nil)

func newHoverIcon(resource fyne.Resource, tooltip *hoverTooltip) *hoverIcon {
	i := &hoverIcon{tooltip: tooltip}
	i.ExtendBaseWidget(i)
	i.SetResource(resource)

	return i
}

func (i *hoverIcon) MouseIn(*desktop.MouseEvent) {
	if i.Text != "" {
		i.tooltip.Show(i, i.Text)
	}
}

func (i *hoverIcon) MouseMoved(*desktop.MouseEvent) {}

func (i *hoverIcon) MouseOut() {
	i.tooltip.Hide()
}
