package apperror

type Kind string

const (
	KindInvalid        Kind = "INVALID"
	KindNotFound       Kind = "NOT_FOUND"
	KindConflict       Kind = "CONFLICT"
	KindUnauthorized   Kind = "UNAUTHORIZED"
	KindForbidden      Kind = "FORBIDDEN"
	KindNotImplemented Kind = "NOT_IMPLEMENTED"
	KindInternal       Kind = "INTERNAL"
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(
	kind Kind,
	code string,
	message string,
) *Error {
	return &Error{
		Kind:    kind,
		Code:    code,
		Message: message,
	}
}

func Wrap(
	kind Kind,
	code string,
	message string,
	err error,
) *Error {
	return &Error{
		Kind:    kind,
		Code:    code,
		Message: message,
		Err:     err,
	}
}
