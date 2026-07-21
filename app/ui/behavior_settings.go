package ui

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/upload"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type BehaviorSettingPopup struct {
	*Popup
	OptionBox *fyne.Container

	settings      map[config.BehaviorSettingType]config.BehaviorSettingVisualDependency
	settingsBox   *fyne.Container
	settingInputs map[config.BehaviorSettingType]*behaviorSettingInput
}

type behaviorSettingInput struct {
	object fyne.CanvasObject
	value  func() (any, error)
	reset  func(any) error
}

func orderedBehaviorSettingTypes(settings map[config.BehaviorSettingType]config.BehaviorSettingVisualDependency) []config.BehaviorSettingType {
	logger.FuncDebug()

	isMobile := runtime.GOOS == "android" || runtime.GOOS == "ios"

	settingTypes := make([]config.BehaviorSettingType, 0, len(settings))
	for settingType, visual := range settings {
		if isMobile && !visual.IsMobile {
			continue
		}

		settingTypes = append(settingTypes, settingType)
	}

	sort.Slice(settingTypes, func(i, j int) bool {
		return settings[settingTypes[i]].Position < settings[settingTypes[j]].Position
	})

	return settingTypes
}

func newBehaviorSettingInput(visual config.BehaviorSettingVisualDependency, onChange func()) (*behaviorSettingInput, error) {
	logger.FuncDebug()

	validate := func(value any) error {
		if visual.Validate != nil {
			return visual.Validate(value)
		}

		return nil
	}

	switch visual.Kind {
	case config.SettingKindBool:
		value, ok := visual.Setting.GetAny().(bool)
		if !ok {
			return nil, fmt.Errorf("setting %q is not a bool", visual.Name)
		}

		check := widget.NewCheck(visual.Name, func(bool) { onChange() })
		check.SetChecked(value)

		return &behaviorSettingInput{
			object: check,
			value:  func() (any, error) { return check.Checked, validate(check.Checked) },
			reset: func(value any) error {
				checked, ok := value.(bool)
				if !ok {
					return fmt.Errorf("setting %q is not a bool", visual.Name)
				}

				check.SetChecked(checked)
				return nil
			},
		}, nil

	case config.SettingKindDuration:
		value, ok := visual.Setting.GetAny().(time.Duration)
		if !ok {
			return nil, fmt.Errorf("setting %q is not a duration", visual.Name)
		}

		entry := widget.NewEntry()
		entry.SetText(value.String())
		entry.Validator = func(text string) error {
			duration, err := time.ParseDuration(text)
			if err != nil {
				return fmt.Errorf("use duration such as 45m or 1h30m")
			}

			return validate(duration)
		}

		return &behaviorSettingInput{
			object: widget.NewForm(widget.NewFormItem(visual.Name, entry)),
			value: func() (any, error) {
				if err := entry.Validate(); err != nil {
					return nil, err
				}

				return time.ParseDuration(entry.Text)
			},
			reset: func(value any) error {
				duration, ok := value.(time.Duration)
				if !ok {
					return fmt.Errorf("setting %q is not a duration", visual.Name)
				}

				entry.SetText(duration.String())
				return nil
			},
		}, nil

	case config.SettingKindInt:
		value, ok := visual.Setting.GetAny().(int)
		if !ok {
			return nil, fmt.Errorf("setting %q is not an int", visual.Name)
		}

		entry := widget.NewEntry()
		entry.SetText(strconv.Itoa(value))

		entry.Validator = func(text string) error {
			value, err := strconv.Atoi(text)
			if err != nil {
				return fmt.Errorf("use a whole number")
			}

			return validate(value)
		}

		return &behaviorSettingInput{
			object: widget.NewForm(widget.NewFormItem(visual.Name, entry)),
			value: func() (any, error) {
				if err := entry.Validate(); err != nil {
					return nil, err
				}

				return strconv.Atoi(entry.Text)
			},
			reset: func(value any) error {
				integer, ok := value.(int)
				if !ok {
					return fmt.Errorf("setting %q is not an int", visual.Name)
				}

				entry.SetText(strconv.Itoa(integer))
				return nil
			},
		}, nil

	default:
		return nil, fmt.Errorf("setting %q has unsupported kind %d", visual.Name, visual.Kind)
	}
}

func NewBehaviorSettingPopup(p *Popup) *BehaviorSettingPopup {
	logger.FuncDebug()

	sp := &BehaviorSettingPopup{
		Popup:         p,
		settings:      p.appConfig.BehaviorConfig.GetSettingsMap(),
		settingInputs: make(map[config.BehaviorSettingType]*behaviorSettingInput),
	}

	settingObjects := make([]fyne.CanvasObject, 0, len(sp.settings))
	for _, settingType := range orderedBehaviorSettingTypes(sp.settings) {
		input, err := newBehaviorSettingInput(sp.settings[settingType], func() {
			sp.refreshChildren(settingType)
		})

		if err != nil {
			logger.Rlogger.Error("Failed to create behavior setting input", slog.Any("err", err))
			continue
		}

		sp.settingInputs[settingType] = input
		settingObjects = append(settingObjects, input.object)
	}

	sp.settingsBox = container.NewVBox(settingObjects...)
	for settingType := range sp.settingInputs {
		sp.refreshChildren(settingType)
	}

	saveBtn := widget.NewButton("Save", func() {
		values := make(map[config.BehaviorSettingType]any, len(sp.settingInputs))
		for settingType, input := range sp.settingInputs {
			value, err := input.value()
			if err != nil {
				return
			}
			values[settingType] = value
		}

		for settingType, value := range values {
			setting := sp.settings[settingType].Setting
			if !reflect.DeepEqual(value, setting.GetAny()) {
				if err := setting.SetAny(value); err != nil {
					logger.Rlogger.Error("Failed to save behavior setting", slog.Any("err", err))
					return
				}
			}
		}

		sp.popup.Hide()
	})
	saveBtn.Importance = widget.HighImportance

	settingsScroll := container.NewVScroll(sp.settingsBox)
	bottom := container.NewVBox(widget.NewSeparator(), saveBtn)

	clearCacheBtn := createClearCacheBtn(p)

	var top fyne.CanvasObject
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		top = container.NewVBox(createExportLogBtn(p), clearCacheBtn, widget.NewSeparator())
	} else {
		top = container.NewVBox(clearCacheBtn, widget.NewSeparator())
	}
	content := container.NewBorder(top, bottom, nil, nil, settingsScroll)

	sp.SetContent(content)
	sp.popup.Resize(fyne.NewSize(320, 400))

	return sp
}

func (sp *BehaviorSettingPopup) Show() {
	logger.FuncDebug()

	sp.resetInputs()

	sp.Popup.Show()
}

func (sp *BehaviorSettingPopup) resetInputs() {
	logger.FuncDebug()

	for settingType, input := range sp.settingInputs {
		if err := input.reset(sp.settings[settingType].Setting.GetAny()); err != nil {
			logger.Rlogger.Error("Failed to reset behavior setting input", slog.Any("err", err))
		}
	}

	for settingType := range sp.settingInputs {
		sp.refreshChildren(settingType)
	}
}

func (sp *BehaviorSettingPopup) refreshChildren(settingType config.BehaviorSettingType) {
	logger.FuncDebug()

	parent, hasParent := sp.settingInputs[settingType]
	visual, hasVisual := sp.settings[settingType]
	if !hasParent || !hasVisual || visual.Kind != config.SettingKindBool {
		return
	}

	value, err := parent.value()
	if err != nil {
		return
	}

	visible, ok := value.(bool)
	if !ok {
		return
	}

	for _, childType := range visual.Children {
		if child, ok := sp.settingInputs[childType]; ok {
			if visible {
				child.object.Show()
			} else {
				child.object.Hide()
			}
		}
	}
}

func createClearCacheBtn(p *Popup) *widget.Button {
	logger.FuncDebug()

	btn := widget.NewButton("Clear Match History Cache", func() {
		dialog.ShowConfirm("Clear Match History Cache", "Are you sure you want to delete match history cache ?", func(confirmed bool) {
			if !confirmed {
				return
			}

			for _, website := range p.appConfig.StorageSettings.Get() {
				upload.LoadUploadedCache(website.Name, 0).Clear()
			}
		}, p.parentWindow)
	})

	btn.Importance = widget.DangerImportance

	return btn
}

func createExportLogBtn(p *Popup) *widget.Button {
	logger.FuncDebug()

	return widget.NewButton("Export Logs", func() {
		saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if writer == nil || err != nil {
				return
			}

			_, err = tools.PathExists(constant.Paths.AppLog)
			if err != nil {
				logger.Rlogger.Error("Error checking logs file:", slog.Any("err", err))
				dialog.ShowInformation("Error", "Failed to export logs!\nLogs file not found", p.parentWindow)
				return
			}

			defer writer.Close()
			logData, err := os.ReadFile(constant.Paths.AppLog)
			if err != nil {
				logger.Rlogger.Error("Error reading logs:", slog.Any("err", err))
				dialog.ShowInformation("Error", "Failed to export logs!\nError reading logs file", p.parentWindow)
				return
			}

			_, err = writer.Write(logData)
			if err != nil {
				logger.Rlogger.Error("Error writing of export:", slog.Any("err", err))
				dialog.ShowInformation("Error", "Failed to export logs!\nError writing logs file", p.parentWindow)
				return
			}

			dialog.ShowInformation("Success", "Logs exported successfully!", p.parentWindow)

		}, p.parentWindow)

		saveDialog.SetFileName("rockpload_logs.txt")
		saveDialog.Show()
	})
}
