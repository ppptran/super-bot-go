package core

import (
	"context"
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

// GetMikroTikInfo fetches MikroTik information via SNMP using goroutine
func GetMikroTikInfo(ctx context.Context, resultChan chan<- MikroTikInfo) {
	defer close(resultChan)

	// Initialize SNMP client
	snmp := &gosnmp.GoSNMP{
		Target:    MikroTikIP,
		Port:      161,
		Community: SNMPCommunity,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
		Retries:   3,
	}

	err := snmp.Connect()
	if err != nil {
		resultChan <- MikroTikInfo{Error: err.Error()}
		return
	}
	defer snmp.Conn.Close()

	// OIDs to query
	oids := []string{
		"1.3.6.1.2.1.1.5.0",            // System name
		"1.3.6.1.2.1.25.3.3.1.2.1",     // CPU usage
		"1.3.6.1.2.1.1.3.0",            // Uptime
		"1.3.6.1.2.1.25.2.3.1.6.65536", // RAM used
		"1.3.6.1.2.1.25.2.3.1.5.65536", // RAM total
	}

	result, err := snmp.Get(oids)
	if err != nil {
		resultChan <- MikroTikInfo{Error: err.Error()}
		return
	}

	// Parse results
	info := MikroTikInfo{
		Name:   "N/A",
		CPU:    "0",
		RAM:    "N/A",
		Uptime: "N/A",
	}

	if len(result.Variables) >= 5 {
		// System name
		if result.Variables[0].Value != nil {
			info.Name = string(result.Variables[0].Value.([]byte))
		}

		// CPU usage
		if result.Variables[1].Value != nil {
			info.CPU = fmt.Sprintf("%d", result.Variables[1].Value)
		}

		// Uptime
		if result.Variables[2].Value != nil {
			ticks := result.Variables[2].Value.(uint32)
			info.Uptime = formatUptime(ticks)
		}

		// RAM
		if result.Variables[3].Value != nil && result.Variables[4].Value != nil {
			ramUsed := result.Variables[3].Value.(int)
			ramTotal := result.Variables[4].Value.(int)
			info.RAM = fmt.Sprintf("%d/%d MB", ramUsed/1024, ramTotal/1024)
		}
	}

	resultChan <- info
}

// formatUptime converts SNMP timeticks to human-readable format
func formatUptime(ticks uint32) string {
	seconds := ticks / 100
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dd %02d:%02d", days, hours, minutes)
}
