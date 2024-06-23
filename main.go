package main

import (
	"context"
	"fmt"
	"me/cobble/routes"
	"me/cobble/utils/db"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/swaggo/echo-swagger"

	_ "me/cobble/docs"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a WaitGroup to keep track of running goroutines
	var wg sync.WaitGroup

	// Start the HTTP server
	wg.Add(1)
	go startServer(ctx, &wg)

	// Listen for termination signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-signalCh

	// Start the graceful shutdown process
	fmt.Println("\nGracefully shutting down...")

	// Cancel the context to signal the HTTP server to stop
	cancel()

	// Wait for the HTTP server to finish
	wg.Wait()

	fmt.Println("Shutdown complete.")
}

func startServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	e := echo.New()

	var conn = utils.EstablishConnection()

	utils.CreateTables(conn)

	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			if strings.Contains(c.Request().URL.Path, "docs") {
				return true
			}
			return false
		},
	}))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.Static("/files", "./files")
	e.GET("/docs/*", echoSwagger.WrapHandler)

	// register routes
	routes.RegisterUserRoutes(e)
	routes.RegisterProjectRoutes(e)
	routes.RegisterVersionRoutes(e)

	// start server
	go func() {
		e.Logger.Fatal(e.Start(":1323"))
	}()

	// when server is done, start shutdown
	<-ctx.Done()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	err := e.Shutdown(shutdownCtx)
	conn.Db.Close()
	if err != nil {
		fmt.Printf("Server shutdown error: %s\n", err)
	}
}
