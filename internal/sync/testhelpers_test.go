package sync

import (
	"context"
	"fmt"
	"sort"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

type fakeItemRepo struct {
	items map[string]*domain.Item // keyed by UUID
}

func newFakeItemRepo() *fakeItemRepo {
	return &fakeItemRepo{items: make(map[string]*domain.Item)}
}

func (r *fakeItemRepo) addItem(item *domain.Item) {
	cp := *item
	r.items[item.UUID] = &cp
}

func (r *fakeItemRepo) FindByUUID(_ context.Context, uuid string) (*domain.Item, error) {
	item := r.items[uuid]
	if item == nil {
		return nil, nil
	}
	cp := *item
	return &cp, nil
}

func (r *fakeItemRepo) FindByUserUUID(_ context.Context, userUUID string, opts FindItemsOpts) ([]*domain.Item, error) {
	var result []*domain.Item

	for _, item := range r.items {
		if item.UserUUID != userUUID {
			continue
		}

		if opts.SinceTimestamp > 0 && item.UpdatedAtTimestamp <= opts.SinceTimestamp {
			continue
		}
		if opts.FromTimestamp > 0 && item.UpdatedAtTimestamp < opts.FromTimestamp {
			continue
		}
		if opts.ContentType != "" && (item.ContentType == nil || *item.ContentType != opts.ContentType) {
			continue
		}

		cp := *item
		result = append(result, &cp)
	}

	// Sort by updated_at_timestamp ASC
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAtTimestamp < result[j].UpdatedAtTimestamp
	})

	if opts.Limit > 0 && len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result, nil
}

func (r *fakeItemRepo) Save(_ context.Context, item *domain.Item) error {
	cp := *item
	r.items[item.UUID] = &cp
	return nil
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}

func fakeUUID(n int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", n)
}
