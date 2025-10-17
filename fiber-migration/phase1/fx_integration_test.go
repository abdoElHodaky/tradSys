package main

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"

	"github.com/abdoElHodaky/tradSys/fiber-migration/phase1/poc"
)

func TestFiberFxIntegration(t *testing.T) {
	app := fxtest.New(t,
		fx.Provide(
			zap.NewDevelopment,
		),
		poc.FiberModule,
		fx.Invoke(func(service *poc.FiberService) {
			// Service should be created successfully
			if service == nil {
				t.Fatal("FiberService should not be nil")
			}
		}),
	)

	app.RequireStart()
	app.RequireStop()
}

func TestFiberServiceLifecycle(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	params := poc.FiberServiceParams{
		Logger: logger,
		DB:     nil, // Optional dependency
	}
	
	service := poc.NewFiberService(params)
	if service == nil {
		t.Fatal("FiberService should not be nil")
	}

	// Test service start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// Give the service a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test service stop
	err = service.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop service: %v", err)
	}
}

func BenchmarkFiberServiceCreation(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		params := poc.FiberServiceParams{
			Logger: logger,
			DB:     nil,
		}
		service := poc.NewFiberService(params)
		_ = service
	}
}
