// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: LicenseRef-COMCAST

package metrics

// events
const ()

// errors
const ()

func GetUnknownTagIfEmpty(tag string) string {
	if tag == "" {
		return "unknown"
	}
	return tag
}
