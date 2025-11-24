package v1alpha1

import (
	"strings"
)

const (
	Added    = "Added"
	Modified = "Modified"
	Deleted  = "Deleted"
	Generic  = "Generic"
)

type TypedEvent[T any] struct {
	Type   string
	Object T
}

func CleanEventType(eventType string) string {
	if eventType == "" {
		return ""
	}

	lower := strings.ToLower(eventType)

	return strings.ToUpper(lower[:1]) + lower[1:]
}
