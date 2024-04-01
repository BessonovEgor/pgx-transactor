# espool
Package is aimed to simplify transaction management in your logic.

## General idea
Since `context.Context` is usually passed to execute any requests to database, it can be the perfect object to inject transaction into. Package provides functionality for wrapping your "within transaction" logic with transactional behavior.

## How to use
Let's imagine we have `MyService` where transactional behavior is required and 2 repositories `OrderRepository` and `PaymentRepository` which must create order and payment consistently within single transaction, so if either first or second fail - no data is created at all and transaction is rolled back.

Firstly, for object which requires transactional behavior you can define property with `transactor` interface, so now `esPool` will work with your transaction function:
```go
package service

type Transactor interface {
    WithinTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error
}

type MyService struct {
	esPool espool.Transactor
}
```
Then, your database accessors (repositories in our case) must be embedded with concrete transaction runner interface - `PgxQuerryRunner`. For version `v2` no need to call `Runner` method of transaction runner (it is still possible though). Please, pay attention on `OrderRepository` method `CreateOrder` below:
```go
package repository

type Order struct {
	// Order fields
}

type Payment struct {
	// Payment fields
}

type OrderRepository struct {
	espool.PgxQueryRunner
}

func NewOrderRepository(r espool.PgxQueryRunner) *OrderRepository {
	return &OrderRepository{PgxQueryRunner: r}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, customerID, orderID int) error {
    q := `insert into orders(customer_id,order_id) values($1,$2)`
    _, err := r.trxExecutor.Exec(ctx, q, customerID, orderID)
    if err != nil {
        return err
    }

    return nil
}

type PaymentRepository struct {
	espool.PgxQueryRunner
}

func NewPaymentRepository(r espool.PgxQueryRunner) *PaymentRepository {
	return &PaymentRepository{PgxQueryRunner: r}
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, customerID, price int) error {
    q := `insert into payment(customer_id,amount) values($1,$2)`
    _, err := r.trxExecutor.Exec(ctx, q, customerID, price)
    if err != nil {
        return err
    }

    return nil
}
```
Now, you can use repositories in combination with transactor in `MyService`:
```go
package service

type MyService struct {
	trx espool.Transactor // in this example will be maintained via espool.NewPgxPool(...)
	orderRps OrderRepository // order repository added
	paymentRps PaymentRepository // payment repository added
}

// ...
// omitting NewMyService
// ...

// new method CreateOrderWithItems is added
func (s *MyService) CreateOrderAndPayment(ctx context.Context, order *Order, payment *Payment) error {
	// WithinTransaction injects pgx.Tx into ctx, so txCtx is already with injected transaction
	err := s.trx.WithinTransaction(ctx, func (txCtx context.Context) error {
		var inTxErr error

		// txCtx has pgx.Tx injected - Runner in repository will return pgx.Tx
		inTxErr = s.orderRps.CreateOrder(txCtx, ...)
		if inTxErr != nil {
			// if error - transaction is rolled back
			return inTxErr
		}
		
		// txCtx has pgx.Tx injected - Runner in repository will return pgx.Tx
		inTxErr = s.paymentRps.CreatePayment(txCtx, ...)
		if inTxErr != nil {
			// if error - transaction is rolled back 
			return inTxErr
		}
		
		// transaction is committed if returned error is nil
		return nil
	})
}
```
As you can see from example above, transaction is shared between 2 repositories through context where it is injected

Function `main` implementation:
```go
package main

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    pgxPool, err := pgxpool.New(ctx, "postgres://postgres:12345@localhost:5432/postgres")
    if err != nil{
	    panic(err)	
    }
	
    exec := espool.NewPgxPool(pgxPool) // same as espool.NewPgxPool(pgxPool).WithTracer(tracer.NewNilTracer()), read about tracers below

    payRps := PaymentRepository{trxExecutor: exec}
    orderRps := OrderRepository{trxExecutor: exec}
    s := ShopService{payRps: payRps, orderRps: orderRps, esPool: exec}

    err = s.CreateOrder(ctx, 12500, 123, 105)
    if err != nil{
        panic(err)
    }
}

```

## Metrics tracing
Also, you can use metrics tracers for OpenTelemetry or DataDog, in default case when you create `espool.PgxPool` it will be created with nil tracer `tracer.NilTracer`, but you can specify tracer by using `espool.NewPgxPool(pgxPool).WithTracer(tracer.Tracer)`.

### DataDog tracing
For DataDog tracing, you must initialize the datadog exporter - `initDataDogExporter(...)` first, then create `espool.PgxPool` with `datadogTracer.DataDogTracer` from `gitlab.effective-soft.com/safqa/datadog-tracer` package:
```go
package main

func initDataDogExporter(address,service,env string) (func(), error) {
        ddTracer.Start(ddTracer.WithAgentAddr(address), ddTracer.WithService(service), ddTracer.WithEnv(env))
    return ddTracer.Stop, nil
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    ddExporterStop, err := initDataDogExporter(ddAddress,"service name","environment")
    defer ddExporterStop()
    if err != nil{
	    panic(err)	
    }

    pgxPool, err := pgxpool.New(ctx, "postgres://postgres:12345@localhost:5432/postgres")
    if err != nil{
        panic(err)
    }
	
    exec := espool.NewPgxPool(pgxPool).WithTracer(datadogTracer.NewTracer("service name"))

    payRps := PaymentRepository{trxExecutor: exec}
    orderRps := OrderRepository{trxExecutor: exec}
    s := ShopService{payRps: payRps, orderRps: orderRps, esPool: exec}

    err = s.CreateOrder(ctx, 12500, 123, 105)
    if err != nil{
        panic(err)
    }
}

```

### OpenTelemetry tracing
For OpenTelemetry tracing, you must initialize the selected exporter(Zipkin exporter in my case) - `initZipkinExporter(...)` first, then create `espool.PgxPool` with `openTelemetryTracer.OpenTelemetryTracer` from `gitlab.effective-soft.com/safqa/open-telemetry-tracer` package:
```go
package main

func initZipkinExporter(address string) (*zipkin.Exporter, error) {
	zExporter, err := zipkin.New(address)
	if err != nil {
		return nil, err
	}
	provider := oTelTrace.NewTracerProvider(
		oTelTrace.WithSyncer(zExporter),
		oTelTrace.WithSampler(oTelTrace.AlwaysSample()),
	)
	otel.SetTracerProvider(provider)

	return zExporter, nil
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    zExporter, err := initZipkinExporter(zipkinAddress)
    defer func(zExporter oTelTrace.SpanExporter, ctx context.Context) {
		err := zExporter.Shutdown(ctx)
		if err != nil {
            panic(err)
        }
    }(zExporter, ctx)
    if err != nil{
	    panic(err)	
    }
	
    pgxPool, err := pgxpool.New(ctx, "postgres://postgres:12345@localhost:5432/postgres")
    if err != nil{
        panic(err)
    }
	
    exec := espool.NewPgxPool(pgxPool).WithTracer(openTelemetryTracer.NewTracer(zExporter, "service name"))
    if err != nil{
        panic(err)
    }

    payRps := PaymentRepository{trxExecutor: exec}
    orderRps := OrderRepository{trxExecutor: exec}
    s := ShopService{payRps: payRps, orderRps: orderRps, esPool: exec}

    err = s.CreateOrder(ctx, 12500, 123, 105)
    if err != nil{
        panic(err)
    }
}
```