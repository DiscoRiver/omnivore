package store

import (
	"fmt"
	"github.com/discoriver/omnivore/internal/test"
	"testing"
)

func TestNewStorageSession(t *testing.T) {
	test.InitTestLogger()

	ss := NewStorageSession()
	fmt.Println(ss.BaseDir)
}
