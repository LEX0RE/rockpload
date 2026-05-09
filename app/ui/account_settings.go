package ui

import (
	"strconv"

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

	allAccounts := asp.appConfig.AccountSettings.Get()

	asp.currentSelectedLabel = widget.NewLabelWithStyle("Current Selected Account: None", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	asp.currentUnusedLabel = widget.NewLabelWithStyle("Current Unused Account: None", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	asp.updateLabels()

	topSection := container.NewVBox(
		asp.currentSelectedLabel,
		asp.currentUnusedLabel,
		widget.NewSeparator(),
	)

	asp.btnAdd = widget.NewButtonWithIcon("Add Account", theme.ContentAddIcon(), asp.onAddAccountBtn)

	asp.btnSetSelected = widget.NewButtonWithIcon("Set Selected", theme.ConfirmIcon(), asp.onSetSelectedBtn)
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

	asp.list = widget.NewList(
		func() int { return len(allAccounts) },
		func() fyne.CanvasObject { return widget.NewLabel("Template...") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			name := allAccounts[i].Player.PlayerName
			suffix := ""

			if asp.appConfig.SelectedAccount() == allAccounts[i] && asp.appConfig.UnusedAccount() == allAccounts[i] {
				suffix = " (Selected & Unused)"
			} else if asp.appConfig.SelectedAccount() == allAccounts[i] {
				suffix = " (Selected)"
			} else if asp.appConfig.UnusedAccount() == allAccounts[i] {
				suffix = " (Unused)"
			}

			o.(*widget.Label).SetText(name + suffix)
		},
	)

	asp.list.OnSelected = asp.onSelected
	asp.list.OnUnselected = asp.onUnselected

	content := container.NewBorder(topSection, bottomSection, nil, nil, asp.list)

	asp.SetContent(content)
	asp.popup.Resize(fyne.NewSize(350, 450))

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

func (asp *AccountSettingsPopup) updateLabels() {
	logger.FuncDebug()

	if asp.appConfig.SelectedAccount() == nil {
		asp.currentSelectedLabel.SetText("Current Selected Account: None")
	} else {
		asp.currentSelectedLabel.SetText("Current Selected Account: " + asp.appConfig.SelectedAccount().Player.PlayerName)
	}

	if asp.appConfig.UnusedAccount() == nil {
		asp.currentUnusedLabel.SetText("Current Unused Account: None")
	} else {
		asp.currentUnusedLabel.SetText("Current Unused Account: " + asp.appConfig.UnusedAccount().Player.PlayerName)
	}
}

func (asp *AccountSettingsPopup) updateBtnStates() {
	logger.FuncDebug()

	allAccounts := asp.appConfig.AccountSettings.Get()

	if asp.selectedIndex < 0 || asp.selectedIndex >= len(allAccounts) {
		asp.btnSetSelected.Disable()
		asp.btnSetUnused.Disable()
		asp.btnUnsetUnused.Disable()
		asp.btnDelete.Disable()
		return
	}

	selectedAccount := allAccounts[asp.selectedIndex]

	if asp.appConfig.SelectedAccount() != nil && asp.appConfig.SelectedAccount().Id() == selectedAccount.Id() {
		asp.btnSetSelected.Disable()
	} else {
		asp.btnSetSelected.Enable()
	}

	if asp.appConfig.UnusedAccount() != nil && asp.appConfig.UnusedAccount().Id() == selectedAccount.Id() {
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

	allAccounts := asp.appConfig.AccountSettings.Get()

	if asp.selectedIndex < 0 || asp.selectedIndex >= len(allAccounts) {
		return
	}

	selectedAccount := allAccounts[asp.selectedIndex]
	asp.appConfig.SetSelectedAccount(selectedAccount.Id())

	asp.Reload()
	asp.list.Refresh()
}

func (asp *AccountSettingsPopup) onSetUnusedBtn() {
	logger.FuncDebug()

	allAccounts := asp.appConfig.AccountSettings.Get()

	if asp.selectedIndex < 0 || asp.selectedIndex >= len(allAccounts) {
		return
	}

	selectedAccount := allAccounts[asp.selectedIndex]
	asp.appConfig.SetUnusedAccount(selectedAccount.Id())

	asp.Reload()
	asp.list.Refresh()
}

func (asp *AccountSettingsPopup) onUnsetUnusedBtn() {
	logger.FuncDebug()

	asp.appConfig.SetUnusedAccount(-1)

	asp.Reload()
	asp.list.Refresh()
}

func (asp *AccountSettingsPopup) onAddAccountBtn() {
	logger.FuncDebug()

	_ = asp.appConfig.AddAccount()

	allAccounts := asp.appConfig.AccountSettings.Get()
	if len(allAccounts) == 1 {
		asp.updateLabels()
	}

	asp.Reload()
	asp.list.Refresh()
	asp.list.Select(len(allAccounts) - 1)
}

func (asp *AccountSettingsPopup) onDeleteAccountBtn() {
	logger.FuncDebug()

	allAccounts := asp.appConfig.AccountSettings.Get()

	if asp.selectedIndex < 0 || asp.selectedIndex >= len(allAccounts) {
		return
	}

	if len(allAccounts) <= 1 {
		dialog.ShowInformation("Impossible action", "You cannot delete the last remaining account.", asp.parentWindow)
		return
	}

	accountToDelete := allAccounts[asp.selectedIndex]
	accountName := accountToDelete.Player.PlayerName + " (ID: " + strconv.Itoa(accountToDelete.Id()) + ")"

	dialog.ShowConfirm("Delete", "Are you sure you want to delete the account '"+accountName+"'?", func(confirmed bool) {
		if confirmed {
			asp.appConfig.DeleteAccount(accountToDelete.Id())

			asp.Reload()
			asp.list.UnselectAll()
			asp.list.Refresh()
		}
	}, asp.parentWindow)
}
