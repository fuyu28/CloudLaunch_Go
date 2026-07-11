package services

import "strings"

// ServiceError はサービス層のユースケースエラーを表す。
type ServiceError struct {
	Message string
	Detail  string
}

func (err *ServiceError) Error() string {
	if err == nil {
		return ""
	}
	if strings.TrimSpace(err.Detail) == "" {
		return err.Message
	}
	return err.Message + ": " + err.Detail
}

func newServiceError(message string, detail string) error {
	return &ServiceError{
		Message: message,
		Detail:  detail,
	}
}
