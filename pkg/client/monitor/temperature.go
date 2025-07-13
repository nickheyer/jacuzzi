package monitor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type TemperatureSensor struct {
	ID         string
	Type       string
	Name       string
	TempMilliC int64 // Temperature in millidegrees Celsius
}

func (t *TemperatureSensor) TempCelsius() float64 {
	return float64(t.TempMilliC) / 1000.0
}

type TemperatureMonitor struct {
	hwmonPath string
}

func NewTemperatureMonitor() *TemperatureMonitor {
	return &TemperatureMonitor{
		hwmonPath: "/sys/class/hwmon",
	}
}

func (m *TemperatureMonitor) GetTemperatures() ([]TemperatureSensor, error) {
	var sensors []TemperatureSensor

	// Read hwmon devices
	hwmonDirs, err := filepath.Glob(filepath.Join(m.hwmonPath, "hwmon*"))
	if err != nil {
		return nil, err
	}

	for _, hwmonDir := range hwmonDirs {
		deviceSensors, err := m.readHwmonDevice(hwmonDir)
		if err != nil {
			// Continue with other devices even if one fails
			continue
		}
		sensors = append(sensors, deviceSensors...)
	}

	// Also try to read CPU temperature from thermal zones
	thermalSensors, err := m.readThermalZones()
	if err == nil {
		sensors = append(sensors, thermalSensors...)
	}

	return sensors, nil
}

func (m *TemperatureMonitor) readHwmonDevice(hwmonDir string) ([]TemperatureSensor, error) {
	var sensors []TemperatureSensor

	// Read device name
	deviceName := "Unknown"
	nameFile := filepath.Join(hwmonDir, "name")
	if data, err := os.ReadFile(nameFile); err == nil {
		deviceName = strings.TrimSpace(string(data))
	}

	// Find all temperature input files
	tempFiles, err := filepath.Glob(filepath.Join(hwmonDir, "temp*_input"))
	if err != nil {
		return nil, err
	}

	for _, tempFile := range tempFiles {
		// Extract sensor number from filename
		base := filepath.Base(tempFile)
		parts := strings.Split(base, "_")
		if len(parts) < 2 {
			continue
		}
		sensorNum := strings.TrimPrefix(parts[0], "temp")

		// Read temperature value
		data, err := os.ReadFile(tempFile)
		if err != nil {
			continue
		}
		tempMilliC, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		if err != nil {
			continue
		}

		// Read label if available
		labelFile := filepath.Join(hwmonDir, fmt.Sprintf("temp%s_label", sensorNum))
		label := fmt.Sprintf("%s_temp%s", deviceName, sensorNum)
		if data, err := os.ReadFile(labelFile); err == nil {
			label = strings.TrimSpace(string(data))
		}

		// Determine sensor type
		sensorType := "OTHER"
		lowerLabel := strings.ToLower(label)
		lowerDevice := strings.ToLower(deviceName)
		if strings.Contains(lowerLabel, "cpu") || strings.Contains(lowerDevice, "coretemp") {
			sensorType = "CPU"
		} else if strings.Contains(lowerLabel, "gpu") || strings.Contains(lowerDevice, "amdgpu") || strings.Contains(lowerDevice, "nvidia") {
			sensorType = "GPU"
		} else if strings.Contains(lowerLabel, "nvme") || strings.Contains(lowerDevice, "nvme") {
			sensorType = "DISK"
		}

		sensor := TemperatureSensor{
			ID:         fmt.Sprintf("%s_%s", filepath.Base(hwmonDir), sensorNum),
			Type:       sensorType,
			Name:       label,
			TempMilliC: tempMilliC,
		}
		sensors = append(sensors, sensor)
	}

	return sensors, nil
}

func (m *TemperatureMonitor) readThermalZones() ([]TemperatureSensor, error) {
	var sensors []TemperatureSensor

	thermalDirs, err := filepath.Glob("/sys/class/thermal/thermal_zone*")
	if err != nil {
		return nil, err
	}

	for _, thermalDir := range thermalDirs {
		// Read temperature
		tempFile := filepath.Join(thermalDir, "temp")
		data, err := os.ReadFile(tempFile)
		if err != nil {
			continue
		}
		tempMilliC, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		if err != nil {
			continue
		}

		// Read type
		typeFile := filepath.Join(thermalDir, "type")
		zoneType := "thermal"
		if data, err := os.ReadFile(typeFile); err == nil {
			zoneType = strings.TrimSpace(string(data))
		}

		sensor := TemperatureSensor{
			ID:         filepath.Base(thermalDir),
			Type:       "CPU",
			Name:       zoneType,
			TempMilliC: tempMilliC,
		}
		sensors = append(sensors, sensor)
	}

	return sensors, nil
}

// ReadCPUInfo reads /proc/cpuinfo to get CPU information
func ReadCPUInfo() (string, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "Unknown CPU", nil
}
