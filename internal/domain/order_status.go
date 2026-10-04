package domain

type OrderStatus string

const OrderStatusCount = 6

const (
	OrderStatusReceived       OrderStatus = "received"
	OrderStatusPreparing      OrderStatus = "preparing"
	OrderStatusReady          OrderStatus = "ready"
	OrderStatusOutForDelivery OrderStatus = "out_for_delivery"
	OrderStatusDelivered      OrderStatus = "delivered"
	OrderStatusCompleted      OrderStatus = "completed"
)

func OrderStatuses() [OrderStatusCount]OrderStatus {
	return [OrderStatusCount]OrderStatus{
		OrderStatusReceived,
		OrderStatusPreparing,
		OrderStatusReady,
		OrderStatusOutForDelivery,
		OrderStatusDelivered,
		OrderStatusCompleted,
	}
}

func IsValidOrderStatus(status string) bool {
	for _, validStatus := range OrderStatuses() {
		if status == string(validStatus) {
			return true
		}
	}
	return false
}
