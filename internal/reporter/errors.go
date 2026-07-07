package reporter

import "fmt"

type ErrorCode string

const (
	ErrMissingRequired     ErrorCode = "MISSING_REQUIRED"
	ErrTypeMismatch        ErrorCode = "TYPE_MISMATCH"
	ErrInvalidEnumValue    ErrorCode = "INVALID_ENUM_VALUE"
	ErrUnsupportedResource ErrorCode = "UNSUPPORTED_RESOURCE"
	ErrNullableViolation   ErrorCode = "NULLABLE_VIOLATION"
	ErrUnexpectedField     ErrorCode = "UNEXPECTED_FIELD"
	ErrUnknown             ErrorCode = "UNKNOWN"
)

var errorTemplates = map[ErrorCode]string{
	ErrMissingRequired:     "Обязательное поле '%s' отсутствует. Ожидался тип: %s.",
	ErrTypeMismatch:        "Несовпадение типа поля '%s': ожидался %s, получен %s.",
	ErrInvalidEnumValue:    "Значение '%s' поля '%s' недопустимо. Ожидалось одно из: %v.",
	ErrUnsupportedResource: "Ресурс '%s' не поддерживается спецификацией.",
	ErrNullableViolation:   "Поле '%s' не может принимать null.",
	ErrUnexpectedField:     "Поле '%s' отсутствует в спецификации.",
}

func FormatErrorMessage(code ErrorCode, args ...interface{}) string {
	tmpl, ok := errorTemplates[code]
	if !ok {
		return fmt.Sprintf("Неизвестная ошибка: код=%s", code)
	}
	return fmt.Sprintf(tmpl, args...)
}
