package example

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	datadogTracer "gitlab.effective-soft.com/safqa/datadog-tracer"
	"gitlab.effective-soft.com/safqa/espool"
	openTelemetryTracer "gitlab.effective-soft.com/safqa/open-telemetry-tracer"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	oTelTrace "go.opentelemetry.io/otel/sdk/trace"
	ddTracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const (
	service       = "espool"
	ddAddress     = "localhost:8126"
	env           = "dev"
	zipkinAddress = "http://localhost:9411/api/v2/spans"
)

func initDataDogExporter(address, service, env string) (func(), error) {
	ddTracer.Start(ddTracer.WithAgentAddr(address), ddTracer.WithService(service), ddTracer.WithEnv(env))
	return ddTracer.Stop, nil
}

func initZipkinExporter(address string) (oTelTrace.SpanExporter, error) {
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

func Test_EsPoolOTelZipkin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zExporter, err := initZipkinExporter(zipkinAddress)
	defer func(zExporter oTelTrace.SpanExporter, ctx context.Context) {
		err := zExporter.Shutdown(ctx)
		if err != nil {
			panic(err)
		}
	}(zExporter, ctx)
	require.NoError(t, err)

	pgxPool, err := pgxpool.New(ctx, "postgres://postgres:12345@localhost:5432/postgres")
	require.NoError(t, err)

	_, err = pgxPool.Exec(ctx, "CREATE TABLE IF NOT EXISTS payment(customer_id INTEGER,amount INTEGER)")
	require.NoError(t, err)

	_, err = pgxPool.Exec(ctx, "CREATE TABLE IF NOT EXISTS orders(customer_id INTEGER,order_id INTEGER)")
	require.NoError(t, err)

	exec := espool.NewPgxPool(pgxPool).WithTracer(openTelemetryTracer.NewTracer(zExporter, service))
	require.NoError(t, err)

	payRps := PaymentRepository{trxExecutor: exec}
	orderRps := OrderRepository{trxExecutor: exec}
	s := ShopService{payRps: payRps, orderRps: orderRps, esPool: exec}

	err = s.CreateOrder(ctx, 12500, 123, 105)
	require.NoError(t, err)
}

func Test_EsPoolDDTrace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ddExporterStop, err := initDataDogExporter(ddAddress, service, env)
	defer ddExporterStop()
	require.NoError(t, err)

	pgxPool, err := pgxpool.New(ctx, "postgres://postgres:12345@localhost:5432/postgres")
	require.NoError(t, err)

	_, err = pgxPool.Exec(ctx, "CREATE TABLE IF NOT EXISTS payment(customer_id INTEGER,amount INTEGER)")
	require.NoError(t, err)

	exec := espool.NewPgxPool(pgxPool).WithTracer(datadogTracer.NewTracer(service))
	require.NoError(t, err)

	payRps := PaymentRepository{trxExecutor: exec}
	orderRps := OrderRepository{trxExecutor: exec}
	s := ShopService{payRps: payRps, orderRps: orderRps, esPool: exec}

	err = s.CreateOrder(ctx, 12500, 123, 105)
	require.NoError(t, err)
}

func Test_EsPoolNoDataTrace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgxPool, err := pgxpool.New(ctx, "postgres://postgres:12345@localhost:5432/postgres")
	require.NoError(t, err)

	_, err = pgxPool.Exec(ctx, "CREATE TABLE IF NOT EXISTS payment(customer_id INTEGER,amount INTEGER)")
	require.NoError(t, err)

	exec := espool.NewPgxPool(pgxPool)
	require.NoError(t, err)

	payRps := PaymentRepository{trxExecutor: exec}
	orderRps := OrderRepository{trxExecutor: exec}
	s := ShopService{payRps: payRps, orderRps: orderRps, esPool: exec}

	err = s.CreateOrder(ctx, 12500, 123, 105)
	require.NoError(t, err)
}
