package config

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type SettingKind uint8

const (
	SettingKindBool SettingKind = iota
	SettingKindDuration
	SettingKindInt
)

type AnySetting interface {
	GetAny() any
	SetAny(value any) error
}

type Setting[T any] struct {
	value    T
	onChange func(T)
	onSet    func()
}

func NewSetting[T any](defaultVal T, onSetHook func()) Setting[T] {
	logger.FuncDebug()

	return Setting[T]{
		value: defaultVal,
		onSet: onSetHook,
	}
}

func (s Setting[T]) MarshalJSON() ([]byte, error) {
	logger.FuncDebug()

	return json.Marshal(s.value)
}

func (s *Setting[T]) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	return json.Unmarshal(data, &s.value)
}

func (s *Setting[T]) Get() T {
	logger.FuncDebug()

	return s.value
}

func (s *Setting[T]) Set(val T) {
	logger.FuncDebug()

	s.value = val

	if s.onSet != nil {
		s.onSet()
	}

	if s.onChange != nil {
		s.onChange(val)
	}
}

func (s *Setting[T]) Bind(callback func(T)) {
	logger.FuncDebug()

	s.onChange = callback
}

func (s *Setting[T]) secretValue() reflect.Value {
	return reflect.ValueOf(&s.value).Elem()
}

func (s *Setting[T]) GetAny() any {
	logger.FuncDebug()

	return s.value
}

func (s *Setting[T]) SetAny(value any) error {
	logger.FuncDebug()

	typedValue, ok := value.(T)
	if !ok {
		return fmt.Errorf(
			"invalid setting value type: expected %T, got %T",
			s.value,
			value,
		)
	}

	s.Set(typedValue)
	return nil
}
