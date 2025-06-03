package hyperliquid

import (
	"testing"
)

func TestConvert_SizeToWire(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		szDec    int
		expected string
	}{
		{
			name:     "BTC Size",
			input:    0.1,
			szDec:    5,
			expected: "0.1",
		},
		{
			name:     "PNUT Size",
			input:    101.22,
			szDec:    1,
			expected: "101.2",
		},
		{
			name:     "ETH Size",
			input:    0.1,
			szDec:    4,
			expected: "0.1",
		},
		{
			name:     "ADA Size",
			input:    100.123456,
			szDec:    0,
			expected: "100",
		},
		{
			name:     "ETH Size",
			input:    1.0,
			szDec:    4,
			expected: "1",
		},
		{
			name:     "ETH Size",
			input:    10.0,
			szDec:    4,
			expected: "10",
		},
		{
			name:     "ETH Size",
			input:    0.0100,
			szDec:    4,
			expected: "0.01",
		},
		{
			name:     "ETH Size",
			input:    0.010000001,
			szDec:    4,
			expected: "0.01",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := SizeToWire(tc.input, tc.szDec)
			if res != tc.expected {
				t.Errorf("SizeToWire() = %v, want %v", res, tc.expected)
			}
		})
	}
}

func TestConvert_PriceToWire(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		maxDec   int
		szDec    int
		expected string
	}{
		{
			name:     "BTC Price",
			input:    105000,
			maxDec:   6,
			szDec:    5,
			expected: "105000",
		},
		{
			name:     "BTC Price",
			input:    105000.1234,
			maxDec:   6,
			szDec:    5,
			expected: "105000",
		},
		{
			name:     "BTC Price",
			input:    95001.123456,
			maxDec:   6,
			szDec:    5,
			expected: "95001",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := PriceToWire(tc.input, tc.maxDec, tc.szDec)
			if res != tc.expected {
				t.Errorf("PriceToWire() = %v, want %v", res, tc.expected)
			}
		})
	}
}
