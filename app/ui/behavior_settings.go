package ui

import (
	"slices"
	"sort"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type BehaviorSettingPopup struct {
	*Popup
	OptionBox *fyne.Container

	optionCheckGroup *widget.CheckGroup

	optionNameMapping map[string]config.BehaviorSettingType
}

func behaviorOptionTextList() []string {
	options := make([]config.BehaviorSettingVisualDependancy, 0, len(config.BehaviorSettingVisualMapping))
	for _, value := range config.BehaviorSettingVisualMapping {
		options = append(options, value)
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Position < options[j].Position
	})

	optionTextList := make([]string, 0, len(options))
	for _, value := range options {
		optionTextList = append(optionTextList, value.Name)
	}

	return optionTextList
}

func NewBehaviorSettingPopup(p *Popup) *BehaviorSettingPopup {
	logger.FuncDebug()

	sp := &BehaviorSettingPopup{Popup: p, optionNameMapping: make(map[string]config.BehaviorSettingType)}

	sp.optionCheckGroup = widget.NewCheckGroup(behaviorOptionTextList(), sp.reload)

	for key, value := range config.BehaviorSettingVisualMapping {
		sp.optionNameMapping[value.Name] = key

		if settingValue, ok := sp.appConfig.BehaviorConfig.GetBoolSettingsMap()[key]; ok && settingValue.Get() {
			sp.optionCheckGroup.Selected = append(sp.optionCheckGroup.Selected, value.Name)
		}
	}

	sp.onCheckGroupChange(sp.optionCheckGroup.Selected)

	saveBtn := widget.NewButton("Save", func() {
		boolSettingsMap := sp.appConfig.BehaviorConfig.GetBoolSettingsMap()

		for settingType, settingValue := range boolSettingsMap {
			for optionName, nameSettingType := range sp.optionNameMapping {
				if settingType == nameSettingType {
					optionValue := slices.Contains(sp.optionCheckGroup.Selected, optionName)

					if optionValue != settingValue.Get() {
						settingValue.Set(optionValue)
					}

					break
				}
			}
		}

		sp.popup.Hide()
	})
	saveBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		sp.optionCheckGroup,
		layout.NewSpacer(),
		saveBtn,
	)

	sp.SetContent(content)

	return sp
}

func (sp *BehaviorSettingPopup) Show() {
	logger.FuncDebug()

	sp.Popup.Show()
}

func (sp *BehaviorSettingPopup) reload(nextSelection []string) {
	sp.optionCheckGroup.Options = behaviorOptionTextList()
	sp.onCheckGroupChange(nextSelection)
}

func (sp *BehaviorSettingPopup) onCheckGroupChange(nextSelection []string) {
	for _, option := range sp.optionCheckGroup.Options {
		optionValue := slices.Contains(nextSelection, option)

		if settingType, ok := sp.optionNameMapping[option]; ok {
			if setting, ok := config.BehaviorSettingVisualMapping[settingType]; ok && len(setting.Children) > 0 {

				for _, childType := range setting.Children {
					if child, ok := config.BehaviorSettingVisualMapping[childType]; ok {
						if optionValue && !slices.Contains(sp.optionCheckGroup.Options, child.Name) {
							sp.optionCheckGroup.Append(child.Name)
						} else if !optionValue && slices.Contains(sp.optionCheckGroup.Options, child.Name) {
							sp.optionCheckGroup.Remove(child.Name)
						}
					}
				}
			}
		}
	}
}
