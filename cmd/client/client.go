package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nickheyer/jacuzzi/client/config"
	"github.com/nickheyer/jacuzzi/client/monitor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	jacuzziv1 "github.com/nickheyer/jacuzzi/proto/gen/go/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "jacuzzi-client",
		Short: "Jacuzzi temperature monitoring client",
		Long:  `Jacuzzi client daemon that monitors hardware temperatures and reports to the server.`,
		RunE:  runClient,
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	// Config file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jacuzzi/client.yaml)")

	// Server flags
	rootCmd.Flags().String("server", "localhost:50051", "The server address")
	rootCmd.Flags().Duration("timeout", 10*time.Second, "Connection timeout")

	// Client flags
	rootCmd.Flags().String("client-id", "", "Client ID (defaults to hostname)")
	rootCmd.Flags().Duration("interval", 30*time.Second, "Temperature reading interval")

	// Monitoring flags
	rootCmd.Flags().Bool("monitor-cpu", true, "Monitor CPU temperatures")
	rootCmd.Flags().Bool("monitor-gpu", true, "Monitor GPU temperatures")
	rootCmd.Flags().Bool("monitor-disk", true, "Monitor disk temperatures")

	// Bind flags to viper
	viper.BindPFlag("server.address", rootCmd.Flags().Lookup("server"))
	viper.BindPFlag("server.timeout", rootCmd.Flags().Lookup("timeout"))
	viper.BindPFlag("client.id", rootCmd.Flags().Lookup("client-id"))
	viper.BindPFlag("client.interval", rootCmd.Flags().Lookup("interval"))
	viper.BindPFlag("monitoring.cpu", rootCmd.Flags().Lookup("monitor-cpu"))
	viper.BindPFlag("monitoring.gpu", rootCmd.Flags().Lookup("monitor-gpu"))
	viper.BindPFlag("monitoring.disk", rootCmd.Flags().Lookup("monitor-disk"))
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

func runClient(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set client ID
	clientID := cfg.Client.ID
	if clientID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to get hostname: %w", err)
		}
		clientID = hostname
	}

	// Connect to server
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Server.Address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer conn.Close()

	client := jacuzziv1.NewTemperatureServiceClient(conn)
	tempMonitor := monitor.NewTemperatureMonitor()

	log.Printf("Starting temperature monitoring client (ID: %s)", clientID)
	log.Printf("Reporting to server: %s", cfg.Server.Address)
	log.Printf("Update interval: %s", cfg.Client.Interval)
	log.Printf("Monitoring: CPU=%v, GPU=%v, Disk=%v", cfg.Monitoring.CPU, cfg.Monitoring.GPU, cfg.Monitoring.Disk)

	// Main monitoring loop
	ticker := time.NewTicker(cfg.Client.Interval)
	defer ticker.Stop()

	// Run immediately on start
	if err := collectAndSendTemperatures(context.Background(), client, tempMonitor, clientID, cfg); err != nil {
		log.Printf("Error sending temperatures: %v", err)
	}

	for range ticker.C {
		if err := collectAndSendTemperatures(context.Background(), client, tempMonitor, clientID, cfg); err != nil {
			log.Printf("Error sending temperatures: %v", err)
		}
	}

	return nil
}

func collectAndSendTemperatures(ctx context.Context, client jacuzziv1.TemperatureServiceClient, monitor *monitor.TemperatureMonitor, clientID string, cfg *config.Config) error {
	// Collect temperature readings
	sensors, err := monitor.GetTemperatures()
	if err != nil {
		return fmt.Errorf("failed to get temperatures: %w", err)
	}

	if len(sensors) == 0 {
		log.Println("No temperature sensors found")
		return nil
	}

	// Filter sensors based on configuration
	var filteredSensors []monitor.TemperatureSensor
	for _, sensor := range sensors {
		include := false
		switch sensor.Type {
		case "CPU":
			include = cfg.Monitoring.CPU
		case "GPU":
			include = cfg.Monitoring.GPU
		case "DISK":
			include = cfg.Monitoring.Disk
		default:
			// Include other sensor types by default
			include = true
		}
		if include {
			filteredSensors = append(filteredSensors, sensor)
		}
	}

	if len(filteredSensors) == 0 {
		log.Println("No sensors to report after filtering")
		return nil
	}

	// Convert to protobuf format
	readings := make([]*jacuzziv1.TemperatureReading, len(filteredSensors))
	timestamp := timestamppb.Now()

	for i, sensor := range filteredSensors {
		readings[i] = &jacuzziv1.TemperatureReading{
			SensorId:           sensor.ID,
			ClientId:           clientID,
			TemperatureCelsius: sensor.TempCelsius(),
			Timestamp:          timestamp,
			SensorType:         sensor.Type,
			SensorName:         sensor.Name,
		}
		log.Printf("Sensor %s (%s): %.1f°C", sensor.Name, sensor.Type, sensor.TempCelsius())
	}

	// Send to server
	req := &jacuzziv1.SubmitTemperatureRequest{
		Readings: readings,
	}

	resp, err := client.SubmitTemperature(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to submit temperatures: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("server returned failure: %s", resp.Message)
	}

	log.Printf("Successfully sent %d temperature readings", len(readings))
	return nil
}
