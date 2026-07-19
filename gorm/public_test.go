package gw_gorm_test

import (
	"context"
	"testing"

	gw_gorm "github.com/generalworksinc/goutil/gorm"
)

type externalContextCarrier struct {
	ctx context.Context
}

func (c *externalContextCarrier) Context() context.Context       { return c.ctx }
func (c *externalContextCarrier) SetContext(ctx context.Context) { c.ctx = ctx }

func TestAttachScopeAcceptsExternalContextCarrier(t *testing.T) {
	carrier := &externalContextCarrier{ctx: context.Background()}
	gw_gorm.AttachScope(carrier, &gw_gorm.Scope{TenantIds: []string{"tenant-a"}})
	if carrier.Context() == nil {
		t.Fatal("AttachScope must preserve a non-nil context")
	}
}
