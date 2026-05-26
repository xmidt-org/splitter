// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

// events
const ()

// errors
const ()

const (
	unknownTagValue = "unknown"
)

func GetUnknownTagIfEmpty(tag string) string {
	if tag == "" {
		return unknownTagValue
	}
	return tag
}
