package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/temporalio/temporal-cafe/proto"
	filterpb "go.temporal.io/api/filter/v1"
	"go.temporal.io/api/workflowservice/v1"
)

func (h *handlers) getOpenKitchenOrderIDs(ctx context.Context) ([]string, error) {
	var nextPageToken []byte
	var orderIDs []string

	for {
		resp, err := h.temporalClient.ListOpenWorkflow(ctx, &workflowservice.ListOpenWorkflowExecutionsRequest{
			Filters: &workflowservice.ListOpenWorkflowExecutionsRequest_TypeFilter{TypeFilter: &filterpb.WorkflowTypeFilter{
				Name: "KitchenOrder",
			}},
			NextPageToken: nextPageToken,
		})
		if err != nil {
			return orderIDs, err
		}

		for _, we := range resp.Executions {
			orderIDs = append(orderIDs, we.Execution.WorkflowId)
		}

		nextPageToken = resp.NextPageToken
		if len(nextPageToken) == 0 {
			break
		}
	}

	return orderIDs, nil
}

func (h *handlers) getKitchenOrderStatus(ctx context.Context, id string) (KitchenOrder, error) {
	var status proto.KitchenOrderStatus

	q, err := h.temporalClient.QueryWorkflow(
		ctx,
		id,
		"",
		proto.KitchenOrderStatusQuery,
	)
	if err != nil {
		return KitchenOrder{}, err
	}

	err = q.Get(&status)
	if err != nil {
		return KitchenOrder{}, err
	}

	order := KitchenOrder{
		ID:   id,
		Name: status.Name,
		Open: status.Open,
	}
	for _, item := range status.Items {
		status := item.Status.String()
		status = strings.TrimPrefix(status, "BARISTA_ORDER_ITEM_STATUS_")
		status = strings.ToLower(status)
		order.Items = append(order.Items, KitchenOrderItem{
			Name:   item.Name,
			Status: status,
		})
	}

	return order, nil

}

func (h *handlers) handleKitchenOrderList(w http.ResponseWriter, r *http.Request) {
	orderIDs, err := h.getOpenKitchenOrderIDs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var orders []KitchenOrder
	for _, id := range orderIDs {
		order, err := h.getKitchenOrderStatus(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		orders = append(orders, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *handlers) handleKitchenOrderItemStatusUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]
	item := vars["item"]

	line, err := strconv.Atoi(item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s, _ := io.ReadAll(r.Body)
	statusJSON := string(s)
	statusJSON = "BARISTA_ORDER_ITEM_STATUS_" + strings.ToUpper(statusJSON)
	status, ok := proto.KitchenOrderItemStatus_value[statusJSON]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown item status: %s", statusJSON), http.StatusInternalServerError)
		return
	}

	err = h.temporalClient.SignalWorkflow(
		r.Context(),
		id,
		"",
		proto.KitchenOrderItemStatusSignal,
		proto.KitchenOrderItemStatusUpdate{
			Line:   uint32(line),
			Status: proto.KitchenOrderItemStatus(status),
		},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	order, err := h.getKitchenOrderStatus(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
