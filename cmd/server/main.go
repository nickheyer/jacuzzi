package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	jacuzziv1 "github.com/nickheyer/jacuzzi/pkg/gen/go/proto/jacuzzi/v1"
	"github.com/nickheyer/jacuzzi/pkg/server/config"
	"github.com/nickheyer/jacuzzi/pkg/server/db"
	"github.com/nickheyer/jacuzzi/pkg/server/service"
	ui "github.com/nickheyer/jacuzzi/pkg/server/ui/jacuzzi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "jacuzzi-server",
		Short: "Jacuzzi temperature monitoring server",
		Long:  `Jacuzzi is a distributed hardware temperature monitoring system.`,
		RunE:  runServer,
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	// Config file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jacuzzi/server.yaml)")

	// Server flags
	rootCmd.Flags().Int("port", 50051, "The server port")
	rootCmd.Flags().String("host", "", "The server host")
	rootCmd.Flags().Int("http-port", 8080, "The HTTP server port for UI")
	rootCmd.Flags().String("http-host", "", "The HTTP server host for UI")

	// Database flags
	rootCmd.Flags().String("db-type", "sqlite", "Database type (sqlite or postgres)")
	rootCmd.Flags().String("db-host", "localhost", "Database host")
	rootCmd.Flags().Int("db-port", 5432, "Database port")
	rootCmd.Flags().String("db-user", "jacuzzi", "Database user")
	rootCmd.Flags().String("db-password", "", "Database password")
	rootCmd.Flags().String("db-name", "data/db/jacuzzi.db", "Database name")
	rootCmd.Flags().String("db-sslmode", "disable", "Database SSL mode")

	// Bind flags to viper
	viper.BindPFlag("server.port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("server.host", rootCmd.Flags().Lookup("host"))
	viper.BindPFlag("server.http_port", rootCmd.Flags().Lookup("http-port"))
	viper.BindPFlag("server.http_host", rootCmd.Flags().Lookup("http-host"))
	viper.BindPFlag("database.type", rootCmd.Flags().Lookup("db-type"))
	viper.BindPFlag("database.host", rootCmd.Flags().Lookup("db-host"))
	viper.BindPFlag("database.port", rootCmd.Flags().Lookup("db-port"))
	viper.BindPFlag("database.user", rootCmd.Flags().Lookup("db-user"))
	viper.BindPFlag("database.password", rootCmd.Flags().Lookup("db-password"))
	viper.BindPFlag("database.name", rootCmd.Flags().Lookup("db-name"))
	viper.BindPFlag("database.sslmode", rootCmd.Flags().Lookup("db-sslmode"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize database
	dbConfig := db.Config{
		Type:     cfg.Database.Type,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}

	database, err := db.NewDatabase(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register temperature service
	tempService := service.NewTemperatureService(database)
	jacuzziv1.RegisterTemperatureServiceServer(grpcServer, tempService)

	// Register reflection service for easier debugging
	reflection.Register(grpcServer)

	// Start listening
	lis, err := net.Listen("tcp", cfg.GetServerAddress())
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	log.Printf("Starting Jacuzzi server on %s", cfg.GetServerAddress())
	log.Printf("Database: %s (%s)", cfg.Database.Type, cfg.Database.Name)

	// Create HTTP server for UI and gRPC-Web

	// Wrap the gRPC server with gRPC-Web
	grpcWebServer := grpcweb.WrapServer(grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			// Allow all origins in development, restrict in production
			return true
		}))

	// Create a handler that serves both gRPC-Web and static files
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a gRPC-Web request
		if grpcWebServer.IsGrpcWebRequest(r) {
			grpcWebServer.ServeHTTP(w, r)
			return
		}

		// Otherwise serve the embedded UI
		fileSystem := ui.GetFileSystem()
		http.FileServer(fileSystem).ServeHTTP(w, r)
	})

	// Configure HTTP server
	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.HTTPHost, cfg.Server.HTTPPort)
	httpServer := &http.Server{
		Addr:         httpAddr,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down servers...")

		// Shutdown HTTP server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}

		// Shutdown gRPC server
		grpcServer.GracefulStop()
	}()

	// Start serving gRPC
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func main() {
	Execute()
}
