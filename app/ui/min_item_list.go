package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type MinItemsLayout struct {
	MinItems     int
	TemplateItem fyne.CanvasObject
}

func NewMinItemsList(minItems int, length func() int, createItem func() fyne.CanvasObject, updateItem func(widget.ListItemID, fyne.CanvasObject)) (*widget.List, *fyne.Container) {
	logger.FuncDebug()

	list := widget.NewList(length, createItem, updateItem)

	template := createItem()

	listContainer := container.New(&MinItemsLayout{
		MinItems:     minItems,
		TemplateItem: template,
	}, list)

	return list, listContainer
}

func (m *MinItemsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	logger.FuncDebug()

	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}

	itemMin := m.TemplateItem.MinSize()
	listMin := objects[0].MinSize()
	requiredHeight := itemMin.Height * float32(m.MinItems)

	return fyne.NewSize(listMin.Width, fyne.Max(listMin.Height, requiredHeight))
}

func (m *MinItemsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	logger.FuncDebug()

	if len(objects) > 0 {
		objects[0].Resize(size)
		objects[0].Move(fyne.NewPos(0, 0))
	}
}
