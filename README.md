<!-->
SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
SPDX-License-Identifier: Apache-2.0
-->

# What Does This Do?

This application consumes a mix of all kinds of wrp events from a high volume, kafka topic.  Based on the contents of wrp messsages, it publishes (splits) the messages into different topics. 

# Requirements

1. Ordering matters to the consumers.  Therefore, the partition key for the wrp messages should be the same for all events for a given CPE to ensure they are written to the same partition.  The partition key should be unique but consistent across events for the CPE, such as device id / mac address. 

2. Fault tolerance.  If publisher consistently fails due to a network outage, the application should detect that and take action to avoid as much record loss as possible.  The application currently detects network outages and pauses the consumer with retries. (There is an external failover process that is not part of this repository.)

3. Should be as close to deliver "exactly once" as possible.  However, there is some tolerance for duplicates.  Currently the library used for publishing does not support transactions.  It likely will in the future. 

# Testing


# Building and Running Locally