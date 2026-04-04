package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

type SyncRequest struct {
	UserUUID             string             `json:"-"`
	SessionUUID          string             `json:"-"`
	ReadOnly             bool               `json:"-"`
	Items                []domain.ItemHash  `json:"items"`
	SyncToken            string             `json:"sync_token"`
	CursorToken          string             `json:"cursor_token"`
	Limit                int                `json:"limit"`
	ContentType          string             `json:"content_type"`
	APIVersion           string             `json:"api"`
	ComputeIntegrityHash bool               `json:"compute_integrity_hash"`
}

type SyncResponse struct {
	RetrievedItems []domain.ItemHTTPRepresentation `json:"retrieved_items"`
	SavedItems     []domain.ItemHTTPRepresentation `json:"saved_items"`
	Conflicts      []domain.SyncConflict           `json:"conflicts"`
	SyncToken      string                          `json:"sync_token,omitempty"`
	CursorToken    *string                         `json:"cursor_token"`
	IntegrityHash  *string                         `json:"integrity_hash,omitempty"`
}

type Service struct {
	items ItemRepository
}

func NewService(items ItemRepository) *Service {
	return &Service{items: items}
}

func (s *Service) Sync(ctx context.Context, req SyncRequest) (*SyncResponse, error) {
	resp := &SyncResponse{
		RetrievedItems: []domain.ItemHTTPRepresentation{},
		SavedItems:     []domain.ItemHTTPRepresentation{},
		Conflicts:      []domain.SyncConflict{},
	}

	// 1. Save items from client
	var saveResp *SaveItemsResponse
	if len(req.Items) > 0 {
		var err error
		saveResp, err = SaveItems(ctx, s.items, SaveItemsRequest{
			UserUUID:    req.UserUUID,
			Items:       req.Items,
			SessionUUID: req.SessionUUID,
			ReadOnly:    req.ReadOnly,
		})
		if err != nil {
			return nil, fmt.Errorf("saving items: %w", err)
		}

		for _, item := range saveResp.SavedItems {
			resp.SavedItems = append(resp.SavedItems, item.ToHTTPRepresentation())
		}
		resp.Conflicts = append(resp.Conflicts, saveResp.Conflicts...)
	}

	// 2. Retrieve items
	getResp, err := GetItems(ctx, s.items, GetItemsRequest{
		UserUUID:    req.UserUUID,
		SyncToken:   req.SyncToken,
		CursorToken: req.CursorToken,
		Limit:       req.Limit,
		ContentType: req.ContentType,
	})
	if err != nil {
		return nil, fmt.Errorf("getting items: %w", err)
	}

	// 3. Filter out items we just saved (prevent sync doubles)
	savedUUIDs := make(map[string]bool)
	for _, item := range resp.SavedItems {
		savedUUIDs[item.UUID] = true
	}
	// Also filter out items that are in sync_conflict
	for _, conflict := range resp.Conflicts {
		if conflict.ServerItem != nil {
			savedUUIDs[conflict.ServerItem.UUID] = true
		}
	}

	for _, item := range getResp.Items {
		if savedUUIDs[item.UUID] {
			continue
		}
		resp.RetrievedItems = append(resp.RetrievedItems, item.ToHTTPRepresentation())
	}

	// 4. Set sync token
	if saveResp != nil && saveResp.SyncToken != "" {
		resp.SyncToken = saveResp.SyncToken
	} else if len(getResp.Items) > 0 {
		lastItem := getResp.Items[len(getResp.Items)-1]
		resp.SyncToken = domain.EncodeSyncToken(lastItem.UpdatedAtTimestamp + 1)
	} else if req.SyncToken != "" {
		resp.SyncToken = req.SyncToken
	} else {
		resp.SyncToken = domain.EncodeSyncToken(0)
	}

	// 5. Set cursor token
	if getResp.CursorToken != "" {
		resp.CursorToken = &getResp.CursorToken
	}

	// 6. Front-load high priority items for first sync
	if req.SyncToken == "" && req.CursorToken == "" {
		frontLoadPriorityItems(resp)
	}

	// 7. Compute integrity hash if requested
	if req.ComputeIntegrityHash && len(resp.RetrievedItems) > 0 {
		hash := computeIntegrityHash(resp.RetrievedItems)
		resp.IntegrityHash = &hash
	}

	return resp, nil
}

func frontLoadPriorityItems(resp *SyncResponse) {
	priorityTypes := map[string]int{
		"SN|ItemsKey":    0,
		"SN|UserPrefs":   1,
		"SN|Theme":       2,
	}

	sort.SliceStable(resp.RetrievedItems, func(i, j int) bool {
		pi, ok1 := priorityTypes[resp.RetrievedItems[i].ContentType]
		pj, ok2 := priorityTypes[resp.RetrievedItems[j].ContentType]
		if ok1 && !ok2 {
			return true
		}
		if !ok1 && ok2 {
			return false
		}
		if ok1 && ok2 {
			return pi < pj
		}
		return false
	})
}

func computeIntegrityHash(items []domain.ItemHTTPRepresentation) string {
	uuids := make([]string, len(items))
	for i, item := range items {
		uuids[i] = item.UUID
	}
	sort.Strings(uuids)
	data, _ := json.Marshal(uuids)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
