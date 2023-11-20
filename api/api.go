package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/temporalio/temporal-cafe/proto"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/client"
)

type handlers struct {
	temporalClient client.Client
}

func convertItemAPIToProto(item *OrderItem) (*proto.OrderLineItem, error) {
	t := strings.ToUpper(item.Type)
	t = "PRODUCT_TYPE_" + t

	pt, ok := proto.ProductType_value[t]
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", item.Type)
	}

	return &proto.OrderLineItem{
		Type:  proto.ProductType(pt),
		Name:  item.Name,
		Price: item.Price,
		Count: item.Count,
	}, nil
}

func (h *handlers) handleOrdersCreate(w http.ResponseWriter, r *http.Request) {
	var input Order

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var items []*proto.OrderLineItem
	for i := range input.Items {
		item, err := convertItemAPIToProto(&input.Items[i])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		items = append(items, item)
	}

	_, err = h.temporalClient.ExecuteWorkflow(
		r.Context(),
		client.StartWorkflowOptions{
			TaskQueue: "cafe",
		},
		workflows.Order,
		&proto.OrderInput{
			Name:         input.Name,
			Email:        input.Email,
			PaymentToken: "fake",
			Items:        items,
		},
	)

	if err != nil {
		log.Printf("failed to start workflow: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func Router(c client.Client) *mux.Router {
	r := mux.NewRouter()

	h := handlers{temporalClient: c}

	r.HandleFunc("/orders", h.handleOrdersCreate).Methods("POST").Name("orders_create")
	r.HandleFunc("/barista/orders", h.handleBaristaOrderList).Methods("GET").Name("barista_orders_list")
	r.HandleFunc("/barista/orders/{id}/{item}/status", h.handleBaristaOrderItemStatusUpdate).Methods("POST").Name("barista_order_item_status_update")

	r.HandleFunc("/kitchen/orders", h.handleKitchenOrderList).Methods("GET").Name("kitchen_orders_list")
	r.HandleFunc("/kitchen/orders/{id}/{item}/status", h.handleKitchenOrderItemStatusUpdate).Methods("POST").Name("kitchen_order_item_status_update")

	return r
}
