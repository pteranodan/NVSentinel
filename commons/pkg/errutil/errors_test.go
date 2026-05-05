package errutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIsTemporaryError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "K8s Timeout",
			err:      apierrors.NewTimeoutError("too slow", 1),
			expected: true,
		},
		{
			name:     "Permanent Forbidden Error",
			err:      apierrors.NewForbidden(schema.GroupResource{}, "name", errors.New("no")),
			expected: false,
		},
		{
			name:     "Context Canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "Wrapped EOF",
			err:      fmt.Errorf("something went wrong: %w", io.EOF),
			expected: true,
		},
		{
			name:     "Random non-retriable error",
			err:      errors.New("something is broken"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTemporaryError(tt.err); got != tt.expected {
				t.Errorf("IsTemporaryError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
