// admin.go — admin-facing order operations. No owner check: these are guarded
// by staff RBAC at the route layer.
package orders

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// adminSettableStatuses are the shipping statuses an admin may set manually.
// Payment-driven states (paid / payment_failed) and refunded are reached
// through their own flows, not arbitrary edits.
var adminSettableStatuses = map[string]bool{
	StatusProcessing: true,
	StatusShipped:    true,
	StatusDelivered:  true,
	StatusCancelled:  true,
}

func (s *service) AdminList(ctx context.Context, status string, from, to *time.Time, page, pageSize int) (*AdminOrderList, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	rows, total, err := s.repo.ListAll(ctx, status, from, to, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}

	orders := make([]AdminOrderSummary, 0, len(rows))
	for _, row := range rows {
		email, err := s.encryptor.Decrypt(row.EmailEnc)
		if err != nil {
			return nil, fmt.Errorf("decrypt order email: %w", err)
		}
		orders = append(orders, AdminOrderSummary{
			OrderSummary: row.OrderSummary,
			Email:        email,
			IsGuest:      row.IsGuest,
		})
	}

	return &AdminOrderList{Orders: orders, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *service) AdminGet(ctx context.Context, id uuid.UUID) (*OrderResponse, error) {
	return s.LoadByID(ctx, id)
}

// AdminUpdateStatus manually sets an order's shipping status. Only forward
// shipping states (processing/shipped/delivered) and cancellation are allowed;
// use AdminRefund to refund a paid order.
func (s *service) AdminUpdateStatus(ctx context.Context, id uuid.UUID, status string) (*OrderResponse, error) {
	if !adminSettableStatuses[status] {
		return nil, ErrInvalidStatus
	}

	// Cancellation of an already-paid order must go through the refund flow so
	// the charge is reversed and stock restocked — block the silent edit.
	if status == StatusCancelled {
		row, _, _, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if row.Status == StatusPaid {
			return nil, ErrNotCancellable
		}
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}
	return s.LoadByID(ctx, id)
}

// AdminRefund refunds a settled order: reverses the Stripe charge, restocks the
// items, and marks the order refunded. Allowed for paid/processing/shipped/
// delivered orders.
func (s *service) AdminRefund(ctx context.Context, id uuid.UUID) (*OrderResponse, error) {
	row, _, _, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !refundable(row.Status) {
		return nil, ErrNotRefundable
	}
	if s.refunder == nil {
		return nil, ErrRefundUnavailable
	}
	if err := s.refunder.RefundOrder(ctx, id); err != nil {
		return nil, fmt.Errorf("refund order %s: %w", id, err)
	}

	items, err := s.repo.ItemsForRestock(ctx, id)
	if err != nil {
		return nil, err
	}
	err = s.repo.WithTx(ctx, func(tx TxRepo) error {
		for _, it := range items {
			if it.ProductID == nil {
				continue
			}
			if err := tx.IncrementStock(ctx, *it.ProductID, it.Quantity); err != nil {
				return err
			}
		}
		return tx.UpdateStatus(ctx, id, StatusRefunded)
	})
	if err != nil {
		return nil, err
	}
	return s.LoadByID(ctx, id)
}

func refundable(status string) bool {
	switch status {
	case StatusPaid, StatusProcessing, StatusShipped, StatusDelivered:
		return true
	default:
		return false
	}
}
