package example

import (
	"context"
	"gitlab.effective-soft.com/safqa/espool"

	"github.com/sirupsen/logrus"
)

// PaymentRepository test Repository struct.
type PaymentRepository struct {
	trxExecutor espool.PgxQueryRunner
}

// OrderRepository test Repository struct.
type OrderRepository struct {
	trxExecutor espool.PgxQueryRunner
}

// ShopService test Service struct.
type ShopService struct {
	esPool   espool.Transactor
	payRps   PaymentRepository
	orderRps OrderRepository
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, customerID, price int) error {
	q := `insert into payment(customer_id,amount) values($1,$2)`
	_, err := r.trxExecutor.Exec(ctx, q, customerID, price)
	if err != nil {
		return err
	}

	return nil
}

func (r *OrderRepository) CreateOrder(ctx context.Context, customerID, orderID int) error {
	q := `insert into orders(customer_id,order_id) values($1,$2)`
	_, err := r.trxExecutor.Exec(ctx, q, customerID, orderID)
	if err != nil {
		return err
	}

	return nil
}

func (s *ShopService) CreateOrder(ctx context.Context, price, customerID, orderID int) error {
	err := s.esPool.WithinTransaction(ctx, func(txCtx context.Context) error {
		err := s.payRps.CreatePayment(txCtx, customerID, price)
		if err != nil {
			logrus.Info("failed to create payment, rolling back transaction...")
			return err
		}

		err = s.orderRps.CreateOrder(txCtx, customerID, orderID)
		if err != nil {
			logrus.Info("failed to create order, rolling back transaction...")
			return err
		}

		return nil
	})

	return err
}
