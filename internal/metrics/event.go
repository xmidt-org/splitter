// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

// Event represents a generic structured metric event that is not tied to a specific metric
type Event struct {
	Name string
	// Attributes contains the structured key-value pairs associated with the log
	Labels []string
	Value  float64
}
