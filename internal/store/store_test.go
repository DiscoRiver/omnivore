package store

import (
	"github.com/discoriver/omnivore/internal/test"
	"testing"
)

func TestNewStorageSession(t *testing.T) {
	test.InitTestLogger()

	_ = NewStorageSession()
}
