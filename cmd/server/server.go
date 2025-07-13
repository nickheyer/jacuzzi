package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	jacuzziv1 "github.com/nickheyer/jacuzzi/pkg/gen/go/proto/jacuzzi/v1"
	"github.com/nickheyer/jacuzzi/pkg/server/config"
	"github.com/nickheyer/jacuzzi/pkg/server/db"
	"github.com/nickheyer/jacuzzi/pkg/server/service"
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

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		grpcServer.GracefulStop()
	}()

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}
