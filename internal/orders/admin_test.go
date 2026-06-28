package orders

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUpdateStatus_RejectsNonShippingStatus(t *testing.T) {
	repo := newStubRepo()
	svc := newTestService(t, repo, nil)
	id := repo.seedOrder(orderRow{Status: StatusProcessing})

	_, err := svc.AdminUpdateStatus(context.Background(), id, StatusPaid)
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestAdminUpdateStatus_SetsShipped(t *testing.T) {
	repo := newStubRepo()
	svc := newTestService(t, repo, nil)
	id := repo.seedOrder(orderRow{Status: StatusProcessing})

	resp, err := svc.AdminUpdateStatus(context.Background(), id, StatusShipped)
	require.NoError(t, err)
	assert.Equal(t, StatusShipped, resp.Order.Status)
}

func TestAdminUpdateStatus_BlocksCancellingPaidOrder(t *testing.T) {
	repo := newStubRepo()
	svc := newTestService(t, repo, nil)
	id := repo.seedOrder(orderRow{Status: StatusPaid})

	_, err := svc.AdminUpdateStatus(context.Background(), id, StatusCancelled)
	assert.ErrorIs(t, err, ErrNotCancellable)
}

func TestAdminRefund_RejectsUnpaidOrder(t *testing.T) {
	repo := newStubRepo()
	svc := newTestService(t, repo, nil)
	id := repo.seedOrder(orderRow{Status: StatusPendingPayment})

	_, err := svc.AdminRefund(context.Background(), id)
	assert.ErrorIs(t, err, ErrNotRefundable)
}

func TestAdminRefund_RefundsPaidOrder(t *testing.T) {
	repo := newStubRepo()
	refunder := &stubRefunder{}
	svc := newTestServiceWithRefunder(t, repo, nil, refunder)
	id := repo.seedOrder(orderRow{Status: StatusPaid})

	resp, err := svc.AdminRefund(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, StatusRefunded, resp.Order.Status)
	assert.Equal(t, []uuid.UUID{id}, refunder.calls)
}
