package ui

import (
	"slices"

	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AccountSettingsPopup struct {
	*Popup

	selectedIndex int

	currentSelectedLabel *widget.Label
	currentUnusedLabel   *widget.Label
	list                 *widget.List

	btnAdd         *widget.Button
	btnSetSelected *widget.Button
	btnUnsetUnused *widget.Button
	btnSetUnused   *widget.Button
	btnDelete      *widget.Button
}

func NewAccountSettingsPopup(p *Popup) *AccountSettingsPopup {
	logger.FuncDebug()

	asp := &AccountSettingsPopup{Popup: p}
	asp.selectedIndex = -1

	asp.currentSelectedLabel = widget.NewLabelWithStyle("Current Selected Account: None", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	asp.currentUnusedLabel = widget.NewLabelWithStyle("Current Unused Account: None", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	asp.updateLabels()

	topSection := container.NewVBox(
		asp.currentSelectedLabel,
		asp.currentUnusedLabel,
		widget.NewSeparator(),
	)

	asp.btnAdd = widget.NewButtonWithIcon("Add Account", theme.ContentAddIcon(), asp.onAddAccountBtn)

	asp.btnSetSelected = widget.NewButtonWithIcon("Select", theme.ConfirmIcon(), asp.onSetSelectedBtn)
	asp.btnSetSelected.Importance = widget.HighImportance
	asp.btnSetSelected.Disable()

	asp.btnSetUnused = widget.NewButtonWithIcon("Set Unused", theme.ConfirmIcon(), asp.onSetUnusedBtn)
	asp.btnSetUnused.Importance = widget.WarningImportance
	asp.btnSetUnused.Disable()

	asp.btnUnsetUnused = widget.NewButtonWithIcon("Unset Unused", theme.ConfirmIcon(), asp.onUnsetUnusedBtn)
	asp.btnUnsetUnused.Importance = widget.WarningImportance
	asp.btnUnsetUnused.Disable()

	asp.btnDelete = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), asp.onDeleteAccountBtn)
	asp.btnDelete.Importance = widget.DangerImportance
	asp.btnDelete.Disable()

	row1 := container.NewHBox(asp.btnAdd, layout.NewSpacer(), asp.btnDelete)
	row2 := container.NewHBox(asp.btnSetSelected, layout.NewSpacer(), asp.btnSetUnused, layout.NewSpacer(), asp.btnUnsetUnused)

	bottomSection := container.NewVBox(
		widget.NewSeparator(),
		row1,
		row2,
	)

	var listContainer *fyne.Container
	asp.list, listContainer = NewMinItemsList(
		5,
		func() int { return len(asp.getAllAccounts()) },
		func() fyne.CanvasObject { return widget.NewLabel("Template...") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			allAccounts := asp.getAllAccounts()
			name := allAccounts[i].AccountName()
			suffix := ""

			if asp.accountManager.GetSelected() == allAccounts[i] && asp.accountManager.GetUnused() == allAccounts[i] {
				suffix = " (Selected & Unused)"
			} else if asp.accountManager.GetSelected() == allAccounts[i] {
				suffix = " (Selected)"
			} else if asp.accountManager.GetUnused() == allAccounts[i] {
				suffix = " (Unused)"
			}

			o.(*widget.Label).SetText(name + suffix)
		},
	)

	asp.list.OnSelected = asp.onSelected
	asp.list.OnUnselected = asp.onUnselected

	content := container.NewBorder(topSection, bottomSection, nil, nil, listContainer)

	asp.SetContent(content)

	return asp
}

func (asp *AccountSettingsPopup) Show() {
	logger.FuncDebug()

	asp.updateLabels()
	asp.list.UnselectAll()
	asp.list.Refresh()

	asp.Reload()

	asp.Popup.Show()
}

func (asp *AccountSettingsPopup) Reload() {
	logger.FuncDebug()

	asp.updateLabels()
	asp.updateBtnStates()
}

func (asp *AccountSettingsPopup) getAllAccounts() []*rocket_network.Account {
	logger.FuncDebug()

	allAccounts := asp.appConfig.AccountSettings.Get()
	allAccountsList := make([]*rocket_network.Account, 0, len(allAccounts))
	for _, account := range allAccounts {
		allAccountsList = append(allAccountsList, account)
	}

	slices.SortFunc(allAccountsList, func(a, b *rocket_network.Account) int {
		return a.Id() - b.Id()
	})

	return allAccountsList
}

func (asp *AccountSettingsPopup) updateLabels() {
	logger.FuncDebug()

	if asp.accountManager.GetSelected() == nil {
		asp.currentSelectedLabel.SetText("Selected Account: None")
	} else {
		asp.currentSelectedLabel.SetText("Selected Account: " + asp.accountManager.GetSelected().AccountName())
	}

	if asp.accountManager.GetUnused() == nil {
		asp.currentUnusedLabel.SetText("Unused Account: None")
	} else {
		asp.currentUnusedLabel.SetText("Unused Account: " + asp.accountManager.GetUnused().AccountName())
	}
}

func (asp *AccountSettingsPopup) updateBtnStates() {
	logger.FuncDebug()

	allAccounts := asp.getAllAccounts()

	if asp.selectedIndex < 0 || asp.selectedIndex >= len(allAccounts) {
		asp.btnSetSelected.Disable()
		asp.btnSetUnused.Disable()
		asp.btnUnsetUnused.Disable()
		asp.btnDelete.Disable()
		return
	}

	selectedAccount := allAccounts[asp.selectedIndex]

	if asp.accountManager.GetSelected() != nil && asp.accountManager.GetSelected().Id() == selectedAccount.Id() {
		asp.btnSetSelected.Disable()
	} else {
		asp.btnSetSelected.Enable()
	}

	if asp.accountManager.GetUnused() != nil && asp.accountManager.GetUnused().Id() == selectedAccount.Id() {
		asp.btnSetUnused.Disable()
		asp.btnUnsetUnused.Enable()
	} else {
		asp.btnSetUnused.Enable()
		asp.btnUnsetUnused.Disable()
	}

	asp.btnDelete.Enable()
}

func (asp *AccountSettingsPopup) onSelected(id widget.ListItemID) {
	logger.FuncDebug()
	asp.selectedIndex = id
	asp.updateBtnStates()
}

func (asp *AccountSettingsPopup) onUnselected(id widget.ListItemID) {
	logger.FuncDebug()
	asp.selectedIndex = -1
	asp.updateBtnStates()
}

func (asp *AccountSettingsPopup) onSetSelectedBtn() {
	logger.FuncDebug()

	allAccounts := asp.getAllAccounts()

	if asp.selectedIndex < 0 || asp.selectedIndex >= len(allAccounts) {
		return
	}

	selectedAccount := allAccounts[asp.selectedIndex]
	asp.accountManager.SetSelected(selectedAccount.Id())

	asp.Reload()
	asp.list.Refresh()
}

func (asp *AccountSettingsPopup) onSetUnusedBtn() {
	logger.FuncDebug()

	allAccounts := asp.getAllAccounts()

	if asp.selectedIndex < 0 || asp.selectedIndex >= len(allAccounts) {
		return
	}

	selectedAccount := allAccounts[asp.selectedIndex]
	asp.accountManager.SetUnused(selectedAccount.Id())

	asp.Reload()
	asp.list.Refresh()
}

func (asp *AccountSettingsPopup) onUnsetUnusedBtn() {
	logger.FuncDebug()

	asp.accountManager.SetUnused(-1)

	asp.Reload()
	asp.list.Refresh()
}

func (asp *AccountSettingsPopup) onAddAccountBtn() {
	logger.FuncDebug()

	newAccount := asp.accountManager.Add()

	allAccounts := asp.getAllAccounts()
	if len(allAccounts) == 1 {
		asp.updateLabels()
	}

	for i, account := range allAccounts {
		if account.Id() == newAccount.Id() {
			asp.list.Select(i)
			asp.accountManager.SetSelected(account.Id())
			break
		}
	}

	asp.Reload()
	asp.list.Refresh()
}

func (asp *AccountSettingsPopup) onDeleteAccountBtn() {
	logger.FuncDebug()

	allAccounts := asp.getAllAccounts()

	if asp.selectedIndex < 0 || asp.selectedIndex >= len(allAccounts) {
		return
	}

	if len(allAccounts) <= 1 {
		dialog.ShowInformation("Impossible action", "You cannot delete the last remaining account.", asp.parentWindow)
		return
	}

	accountToDelete := allAccounts[asp.selectedIndex]
	dialog.ShowConfirm("Delete", "Are you sure you want to delete the account '"+accountToDelete.AccountName()+"'?", func(confirmed bool) {
		if confirmed {
			asp.accountManager.Delete(accountToDelete.Id())

			asp.Reload()
			asp.list.UnselectAll()
			asp.list.Refresh()
		}
	}, asp.parentWindow)
}
