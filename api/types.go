package api

type MenuItem struct {
	Type  string
	Name  string
	Price uint32
}

type Menu struct {
	Items []MenuItem
}

type OrderItem struct {
	Type  string
	Name  string
	Price uint32
	Count uint32
}

type Order struct {
	Name  string
	Email string

	Items []OrderItem
}

type BaristaOrderItem struct {
	Name   string
	Status string
}

type BaristaOrder struct {
	ID   string
	Name string
	Open bool

	Items []BaristaOrderItem
}

type KitchenOrderItem struct {
	Name   string
	Status string
}

type KitchenOrder struct {
	ID   string
	Name string
	Open bool

	Items []KitchenOrderItem
}
