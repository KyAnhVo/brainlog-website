package memstore

import (
	"testing"

	"github.com/KyAnhVo/brainlog-website/server/internal/store"
	"github.com/KyAnhVo/brainlog-website/server/internal/store/storetest"
)

func TestContract(t *testing.T) {
	storetest.Run(t, func(*testing.T) store.Store { return New() })
}
