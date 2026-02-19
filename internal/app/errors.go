package app

import "github.com/svaruag/mocli/internal/exitcode"

type appError struct {
	Code    string
	Message string
	Hint    string
	Exit    int
}

func (e *appError) Error() string {
	return e.Message
}

func usageError(msg, hint string) error {
	return &appError{Code: "usage_error", Message: msg, Hint: hint, Exit: exitcode.UsageError}
}

func authRequiredError(msg, hint string) error {
	return &appError{Code: "auth_required", Message: msg, Hint: hint, Exit: exitcode.AuthRequired}
}

func permissionError(msg, hint string) error {
	return &appError{Code: "permission_denied", Message: msg, Hint: hint, Exit: exitcode.PermissionDenied}
}

func notFoundError(msg, hint string) error {
	return &appError{Code: "not_found", Message: msg, Hint: hint, Exit: exitcode.NotFound}
}

func transientError(msg, hint string) error {
	return &appError{Code: "transient_error", Message: msg, Hint: hint, Exit: exitcode.TransientError}
}

func notImplementedError(msg, hint string) error {
	return &appError{Code: "not_implemented", Message: msg, Hint: hint, Exit: exitcode.NotImplemented}
}
