package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type CenterWrapLayout struct {
	Spacing float32
}

func NewCenterWrapLayout() fyne.Layout {
	logger.FuncDebug()

	return &CenterWrapLayout{
		Spacing: theme.Padding(),
	}
}

func (c *CenterWrapLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	logger.FuncDebug()

	var rows [][]fyne.CanvasObject
	var currentRow []fyne.CanvasObject
	var currentWidth float32

	for _, obj := range objects {
		if !obj.Visible() {
			continue
		}

		minSize := obj.MinSize()

		if currentWidth+minSize.Width > size.Width && len(currentRow) > 0 {
			rows = append(rows, currentRow)
			currentRow = nil
			currentWidth = 0
		}

		currentRow = append(currentRow, obj)
		if len(currentRow) == 1 {
			currentWidth = minSize.Width
		} else {
			currentWidth += c.Spacing + minSize.Width
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	yPos := float32(0)
	for _, row := range rows {
		var rowWidth float32
		var maxHeight float32

		for i, obj := range row {
			minSize := obj.MinSize()
			if i == 0 {
				rowWidth = minSize.Width
			} else {
				rowWidth += c.Spacing + minSize.Width
			}

			if minSize.Height > maxHeight {
				maxHeight = minSize.Height
			}
		}

		xPos := (size.Width - rowWidth) / 2
		if xPos < 0 {
			xPos = 0
		}

		for _, obj := range row {
			minSize := obj.MinSize()
			obj.Move(fyne.NewPos(xPos, yPos))
			obj.Resize(minSize)
			xPos += minSize.Width + c.Spacing
		}

		yPos += maxHeight + c.Spacing
	}
}

func (c *CenterWrapLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	logger.FuncDebug()

	var minWidth, minHeight float32

	for _, obj := range objects {
		if !obj.Visible() {
			continue
		}

		size := obj.MinSize()
		if size.Width > minWidth {
			minWidth = size.Width
		}
		if size.Height > minHeight {
			minHeight = size.Height
		}
	}

	return fyne.NewSize(minWidth, minHeight)
}
