package core

import (
	"context"
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

// GetPPPoESpeed measures PPPoE bandwidth by sampling SNMP counters twice
func GetPPPoESpeed(ctx context.Context, resultChan chan<- PPPoESpeed) {
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
		resultChan <- PPPoESpeed{Error: err.Error()}
		return
	}
	defer snmp.Conn.Close()

	// OIDs for PPPoE interface counters
	rxOID := fmt.Sprintf("1.3.6.1.2.1.31.1.1.1.6.%s", PPPoEIndex)  // ifHCInOctets
	txOID := fmt.Sprintf("1.3.6.1.2.1.31.1.1.1.10.%s", PPPoEIndex) // ifHCOutOctets

	// Sample 1
	result1, err := snmp.Get([]string{rxOID, txOID})
	if err != nil {
		resultChan <- PPPoESpeed{Error: err.Error()}
		return
	}

	if len(result1.Variables) < 2 {
		resultChan <- PPPoESpeed{Error: "Invalid SNMP response"}
		return
	}

	rx1 := result1.Variables[0].Value.(uint64)
	tx1 := result1.Variables[1].Value.(uint64)

	// Wait 1 second
	time.Sleep(1 * time.Second)

	// Sample 2
	result2, err := snmp.Get([]string{rxOID, txOID})
	if err != nil {
		resultChan <- PPPoESpeed{Error: err.Error()}
		return
	}

	if len(result2.Variables) < 2 {
		resultChan <- PPPoESpeed{Error: "Invalid SNMP response"}
		return
	}

	rx2 := result2.Variables[0].Value.(uint64)
	tx2 := result2.Variables[1].Value.(uint64)

	// Calculate speed in Mbps: (bytes_diff * 8) / 1048576
	rxSpeed := float64(rx2-rx1) * 8 / 1048576
	txSpeed := float64(tx2-tx1) * 8 / 1048576

	resultChan <- PPPoESpeed{
		RxSpeed: roundFloat(rxSpeed, 2),
		TxSpeed: roundFloat(txSpeed, 2),
	}
}

// roundFloat rounds a float to specified decimal places
func roundFloat(val float64, precision int) float64 {
	ratio := float64(1)
	for i := 0; i < precision; i++ {
		ratio *= 10
	}
	return float64(int(val*ratio+0.5)) / ratio
}
