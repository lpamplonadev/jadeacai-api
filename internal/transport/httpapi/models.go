package httpapi

import "github.com/lpamplonadev/jadeacai-bkend/internal/application"

type createOrderRequest = application.CreateOrderRequest
type orderLineRequest = application.OrderLineRequest
type orderCustomer = application.OrderCustomer
type orderAcai = application.OrderAcai
type orderDelivery = application.OrderDelivery
type orderPayment = application.OrderPayment
type orderListFilter = application.OrderListFilter
type storedOrder = application.StoredOrder
type createdOrder = application.CreatedOrder
type paginatedOrders = application.PaginatedOrders
type orderStatusCount = application.OrderStatusCount
type dashboardRecentOrder = application.DashboardRecentOrder
type dashboardData = application.DashboardData

type catalogItemRecord = application.CatalogItemRecord
type catalogRuleRecord = application.CatalogRuleRecord
type catalogComboItemRecord = application.CatalogComboItemRecord
type catalogComboGourmetSizeRecord = application.CatalogComboGourmetSizeRecord
type catalogComboRecord = application.CatalogComboRecord
type catalogData = application.CatalogData
type createCatalogItemRequest = application.CreateCatalogItemRequest
type updateCatalogItemRequest = application.UpdateCatalogItemRequest
type catalogComboItemInput = application.CatalogComboItemInput
type createCatalogComboRequest = application.CreateCatalogComboRequest
type updateCatalogComboRequest = application.UpdateCatalogComboRequest
