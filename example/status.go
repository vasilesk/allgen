package example

// Status is the state of an order.
type Status string

// Statuses an order passes through while it is processed.
//
//go:generate go run github.com/vasilesk/allgen/cmd/allgen -name ActiveStatuses
const (
	StatusNew     Status = "new"
	StatusPaid    Status = "paid"
	StatusShipped Status = "shipped"
)

// Statuses after which an order does not change.
//
//go:generate go run github.com/vasilesk/allgen/cmd/allgen -name FinalStatuses
const (
	StatusDelivered Status = "delivered"
	StatusCanceled  Status = "canceled"
)
