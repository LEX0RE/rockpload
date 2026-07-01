package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type secretSettings map[string]json.RawMessage

type secretSettingValue interface {
	secretValue() reflect.Value
}

func (a *AppConfig) saveSecretSettings() (func(), error) {
	logger.FuncDebug()

	settings := secretSettings{}
	restoreSecretFields, err := extractSecretSettings(reflect.ValueOf(a), nil, settings)
	if err != nil {
		return restoreSecretFields, err
	}

	return restoreSecretFields, tools.SaveJSONFilePath(constant.Paths.SecretSettingsFile, settings)
}

func (a *AppConfig) loadSecretSettings() error {
	logger.FuncDebug()

	settings := secretSettings{}
	if err := tools.LoadJSONFilePath(constant.Paths.SecretSettingsFile, &settings, false); err != nil {
		return err
	}

	return applySecretSettings(reflect.ValueOf(a), nil, settings)
}

// TODO Deprecated, there won't have any migration of previous secret config
func (a *AppConfig) migrateSecretSettings() error {
	logger.FuncDebug()

	publicSettings := secretSettings{}
	restoreSecretFields, err := extractSecretSettings(reflect.ValueOf(a), nil, publicSettings)
	if err != nil {
		restoreSecretFields()
		return err
	}

	settings := secretSettings{}
	if err := tools.LoadJSONFilePath(constant.Paths.SecretSettingsFile, &settings, false); err != nil {
		restoreSecretFields()
		return err
	}

	migrated := false
	cleanPublicSettings := false
	for path, value := range publicSettings {
		if isEmptySecretValue(value) {
			continue
		}

		cleanPublicSettings = true
		if _, exists := settings[path]; exists {
			continue
		}

		settings[path] = value
		migrated = true
	}

	if migrated {
		if err := tools.SaveJSONFilePath(constant.Paths.SecretSettingsFile, settings); err != nil {
			restoreSecretFields()
			return err
		}
	}

	if cleanPublicSettings {
		accountErr, storageErr, behaviorErr := a.savePublicSettings()
		if accountErr != nil || storageErr != nil || behaviorErr != nil {
			restoreSecretFields()
			return fmt.Errorf("failed to clean secret settings from public config: %w", errors.Join(accountErr, storageErr, behaviorErr))
		}
	}

	restoreSecretFields()

	return applySecretSettings(reflect.ValueOf(a), nil, settings)
}

func extractSecretSettings(value reflect.Value, path []string, settings secretSettings) (func(), error) {
	restoreFuncs := []func(){}

	restore := func() {
		for i := len(restoreFuncs) - 1; i >= 0; i-- {
			restoreFuncs[i]()
		}
	}

	err := walkSecretSettings(value, path, func(field reflect.Value, fieldPath []string) error {
		if !field.CanInterface() || !field.CanSet() {
			return fmt.Errorf("secret field %s cannot be read or changed", strings.Join(fieldPath, "."))
		}

		data, err := json.Marshal(field.Interface())
		if err != nil {
			return err
		}

		settings[strings.Join(fieldPath, ".")] = data

		original := reflect.New(field.Type()).Elem()
		original.Set(field)
		restoreFuncs = append(restoreFuncs, func() {
			field.Set(original)
		})

		field.Set(reflect.Zero(field.Type()))
		return nil
	})

	return restore, err
}

func applySecretSettings(value reflect.Value, path []string, settings secretSettings) error {
	return walkSecretSettings(value, path, func(field reflect.Value, fieldPath []string) error {
		data, ok := settings[strings.Join(fieldPath, ".")]
		if !ok {
			return nil
		}
		if !field.CanSet() {
			return fmt.Errorf("secret field %s cannot be changed", strings.Join(fieldPath, "."))
		}

		target := reflect.New(field.Type())
		if err := json.Unmarshal(data, target.Interface()); err != nil {
			return err
		}

		field.Set(target.Elem())
		return nil
	})
}

func walkSecretSettings(value reflect.Value, path []string, onSecretField func(reflect.Value, []string) error) error {
	if !value.IsValid() {
		return nil
	}

	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	if value.CanAddr() {
		if setting, ok := value.Addr().Interface().(secretSettingValue); ok {
			return walkSecretSettings(setting.secretValue(), path, onSecretField)
		}
	}

	switch value.Kind() {
	case reflect.Struct:
		valueType := value.Type()
		for i := range value.NumField() {
			fieldType := valueType.Field(i)
			if fieldType.PkgPath != "" || fieldType.Tag.Get("json") == "-" {
				continue
			}

			field := value.Field(i)
			fieldPath := append(path, jsonFieldName(fieldType))
			if fieldType.Tag.Get("secret") == "true" {
				if err := onSecretField(field, fieldPath); err != nil {
					return err
				}

				continue
			}

			if err := walkSecretSettings(field, fieldPath, onSecretField); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range value.Len() {
			item := value.Index(i)
			if err := walkSecretSettings(item, append(path, listPathSegment(item, i)), onSecretField); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			if err := walkSecretSettings(value.MapIndex(key), append(path, fmt.Sprintf("[%v]", key.Interface())), onSecretField); err != nil {
				return err
			}
		}
	}

	return nil
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}

	name := strings.Split(tag, ",")[0]
	if name == "" {
		return field.Name
	}

	return name
}

func listPathSegment(value reflect.Value, index int) string {
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return fmt.Sprintf("[%d]", index)
		}
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return fmt.Sprintf("[%d]", index)
	}

	valueType := value.Type()
	for i := range value.NumField() {
		fieldType := valueType.Field(i)
		if fieldType.PkgPath != "" || fieldType.Tag.Get("secret_id") != "true" {
			continue
		}

		return fmt.Sprintf("[%s=%v]", jsonFieldName(fieldType), value.Field(i).Interface())
	}

	return fmt.Sprintf("[%d]", index)
}

func isEmptySecretValue(data json.RawMessage) bool {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return false
	}

	switch value := value.(type) {
	case nil:
		return true
	case string:
		return value == ""
	case []any:
		return len(value) == 0
	case map[string]any:
		return len(value) == 0
	}

	return false
}
