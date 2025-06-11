package order

import (
	"Microservices-based-E-commerce-System/account"
	"Microservices-based-E-commerce-System/catalog"
	"Microservices-based-E-commerce-System/order/pb"
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type grpcServer struct {
	pb.UnimplementedOrderServiceServer
	service       Service
	accountClient *account.Client
	catalogClient *catalog.Client
}

func ListenGRPC(s Service, accountURL, catalogURL string, port int) error {
	accountClient, err := account.NewClient(accountURL)
	if err != nil {
		return err
	}
	catalogClient, err := catalog.NewClient(catalogURL)
	if err != nil {
		accountClient.Close()
		return err
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		accountClient.Close()
		catalogClient.Close()
		return err
	}
	serv := grpc.NewServer()
	pb.RegisterOrderServiceServer(serv, &grpcServer{
		UnimplementedOrderServiceServer: pb.UnimplementedOrderServiceServer{},
		service:                         s,
		accountClient:                   accountClient,
		catalogClient:                   catalogClient,
	})
	reflection.Register(serv)
	return serv.Serve(lis)
}

// Handles creation of a new order
func (s *grpcServer) PostOrder(ctx context.Context, r *pb.PostOrderRequest) (*pb.PostOrderResponse, error) {
	// Check if account exists before creating order
	_, err := s.accountClient.GetAccount(ctx, r.AccountId)
	if err != nil {
		log.Println("Error getting account", err)
		return nil, errors.New("account not found")
	}

	// Extract product IDs from request
	// The client only sends productId + quantity, but we need the full product details (name, desc, price)
	// So we extract the list of productIds to fetch details from Catalog service
	productIDs := []string{}
	for _, p := range r.Products {
		productIDs = append(productIDs, p.ProductId)
	}

	// Call Catalog service to get real product details (price, name, desc)
	// Catalog service is the source of truth for product details.
	orderedProducts, err := s.catalogClient.GetProducts(ctx, productIDs, "", 0, 0)
	if err != nil {
		log.Println("Error getting products", err)
		return nil, errors.New("products not found")
	}

	// Build OrderedProduct slice to pass to Order Service
	// We use the product details from Catalog service
	// For each product, we also need to get the correct quantity from the original request
	products := []OrderedProduct{}
	for _, p := range orderedProducts {
		product := OrderedProduct{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
			Quantity:    0, // Initially 0 - we will set correct quantity next
		}
		// Match this product with request product to get correct quantity
		for _, rp := range r.Products {
			if rp.ProductId == p.ID {
				product.Quantity = rp.Quantity
				break
			}
		}
		// Only add products with non-zero quantity
		if product.Quantity != 0 {
			products = append(products, product)
		}
	}

	// Call actual Order Service to create the order
	// Pass in the accountId + correct list of OrderedProducts
	order, err := s.service.PostOrder(ctx, r.AccountId, products)
	if err != nil {
		log.Println("error posting order")
		return nil, errors.New("could not post order")
	}

	// Build the gRPC response - pb.Order
	// We need to convert the internal Go Order model to the proto model pb.Order
	orderProto := &pb.Order{
		Id:         order.ID,
		AccountId:  order.AccountID,
		TotalPrice: order.TotalPrice,
		Products:   []*pb.Order_OrderProduct{},
	}
	orderProto.CreatedAt, _ = order.CreatedAt.MarshalBinary() // Serialize the Go time.Time to binary (bytes) using MarshalBinary

	// Add each product to the proto response
	for _, p := range order.Products {
		orderProto.Products = append(orderProto.Products, &pb.Order_OrderProduct{
			Id:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
			Quantity:    p.Quantity,
		})
	}
	return &pb.PostOrderResponse{
		Order: orderProto,
	}, nil
}

// Fetches the list of orders made by a particular account
func (s *grpcServer) GetOrdersForAccount(ctx context.Context, r *pb.GetOrdersForAccountRequest) (*pb.GetOrdersForAccountResponse, error) {
	// Get all orders for the given account
	// These orders contain only Order ID, Account ID, Total price and Products - id and quanitity only (no product details like name, desc, price).
	accountOrders, err := s.service.GetOrdersForAccount(ctx, r.AccountId)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	// Build a unique list of all product IDs across all the orders.
	// Because we want to fetch full details of a product.
	// We use a MAP here to automatically deduplicate — so we only fetch each product once even if it appears in multiple orders.
	productIDMap := map[string]bool{}
	for _, o := range accountOrders {
		for _, p := range o.Products {
			productIDMap[p.ID] = true
		}
	}
	// Convert the map keys into a simple slice of product IDs, so we can send it to CatalogClient.GetProducts
	productIDs := []string{}
	for id := range productIDMap {
		productIDs = append(productIDs, id)
	}
	// Fetch full product details (Name, Description, Price) for the given productIDs
	products, err := s.catalogClient.GetProducts(ctx, productIDs, "", 0, 0)
	if err != nil {
		log.Println("Error getting account products: ", err)
		return nil, err
	}
	// Construct the response orders for the gRPC response.
	// We need to decorate each product inside each order with its full details fetched above.
	orders := []*pb.Order{}
	for _, o := range accountOrders { // o is each Order from the Order service.
		// Create the base proto.Order object
		op := &pb.Order{
			Id:         o.ID,
			AccountId:  o.AccountID,
			TotalPrice: o.TotalPrice,
			Products:   []*pb.Order_OrderProduct{},
		}
		op.CreatedAt, _ = o.CreatedAt.MarshalBinary()
		// For each product in this order, find its corresponding full details and populate the proto product.
		for _, product := range o.Products {
			// Find the matching product from the list of full Catalog products:
			for _, p := range products { // p is a full product returned by CatalogClient (ID, Name, Desc, Price).
				if p.ID == product.ID {
					product.Name = p.Name
					product.Description = p.Description
					product.Price = p.Price
					break
				}
			}
			// Append the fully decorated product to the proto Order.
			op.Products = append(op.Products, &pb.Order_OrderProduct{
				Id:          product.ID,
				Name:        product.Name,
				Description: product.Description,
				Price:       product.Price,
				Quantity:    product.Quantity,
			})
		}
		orders = append(orders, op)
	}
	return &pb.GetOrdersForAccountResponse{Orders: orders}, nil
}
