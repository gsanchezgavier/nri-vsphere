// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package process

import (
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/nri-vsphere/internal/load"
)

func createDatacenterSamples(config *load.Config) {

	if !config.IsVcenterAPIType {
		return
	}
	for _, dc := range config.Datacenters {

		//Hosts
		var totalMemoryHost int64
		var totalMemoryUsedHost int32
		var totalCpuHost int16
		var totalMHz float64
		var cpuOverallUsage float64

		//Datastore
		var totalDatastoreCapacity int64
		var totalDatastoreFreeSpace int64

		//ResourcePools
		var countResourcePools int64

		//Creating entity name
		datacenterName := dc.Datacenter.Name
		entityName := sanitizeEntityName(config, datacenterName, "")
		uniqueIdentifier := entityName
		ms, err := createNewEntityWithMetricSet(config, entityTypeDatacenter, entityName, uniqueIdentifier)
		if err != nil {
			config.Logrus.WithError(err).WithField("datacenterName", entityName).WithField("uniqueIdentifier", uniqueIdentifier).Error("failed to create metricSet")
			continue
		}

		for _, datastore := range dc.Datastores {
			totalDatastoreCapacity = totalDatastoreCapacity + datastore.Summary.Capacity
			totalDatastoreFreeSpace = totalDatastoreFreeSpace + datastore.Summary.FreeSpace
		}

		for _, resourcePool := range dc.ResourcePools {
			if dc.IsDefaultResourcePool(resourcePool.Reference()) {
				continue
			}
			countResourcePools++
		}

		for _, host := range dc.Hosts {
			if host.Summary.Hardware == nil {
				continue
			}
			totalMHz = totalMHz + (float64(host.Summary.Hardware.CpuMhz) * float64(host.Summary.Hardware.NumCpuCores))
			cpuOverallUsage = cpuOverallUsage + float64(host.Summary.QuickStats.OverallCpuUsage)
			totalCpuHost = totalCpuHost + host.Summary.Hardware.NumCpuCores
			totalMemoryHost = totalMemoryHost + host.Summary.Hardware.MemorySize/(1<<20)
			totalMemoryUsedHost = totalMemoryUsedHost + host.Summary.QuickStats.OverallMemoryUsage
		}

		if totalMHz != 0 {
			cpuPercentHost := cpuOverallUsage / totalMHz * 100
			checkError(config, ms.SetMetric("cpu.overallUsagePercentage", cpuPercentHost, metric.GAUGE))
		}

		if totalMemoryHost != 0 {
			memoryPercentHost := float64(totalMemoryUsedHost) / float64(totalMemoryHost) * 100
			checkError(config, ms.SetMetric("mem.usagePercentage", memoryPercentHost, metric.GAUGE))
		}

		checkError(config, ms.SetMetric("mem.size", totalMemoryHost, metric.GAUGE))
		checkError(config, ms.SetMetric("mem.usage", totalMemoryUsedHost, metric.GAUGE))
		checkError(config, ms.SetMetric("cpu.cores", totalCpuHost, metric.GAUGE))
		checkError(config, ms.SetMetric("cpu.overallUsage", cpuOverallUsage, metric.GAUGE))
		checkError(config, ms.SetMetric("cpu.totalMHz", totalMHz, metric.GAUGE))

		checkError(config, ms.SetMetric("datastore.totalGiB", totalDatastoreCapacity/(1<<30), metric.GAUGE))
		checkError(config, ms.SetMetric("datastore.totalFreeGiB", totalDatastoreFreeSpace/(1<<30), metric.GAUGE))
		checkError(config, ms.SetMetric("datastore.totalUsedGiB", (totalDatastoreCapacity-totalDatastoreFreeSpace)/(1<<30), metric.GAUGE))

		checkError(config, ms.SetMetric("overallStatus", string(dc.Datacenter.OverallStatus), metric.ATTRIBUTE))
		checkError(config, ms.SetMetric("datastores", len(dc.Datastores), metric.GAUGE))
		checkError(config, ms.SetMetric("hostCount", len(dc.Hosts), metric.GAUGE))
		checkError(config, ms.SetMetric("vmCount", len(dc.VirtualMachines), metric.GAUGE))
		checkError(config, ms.SetMetric("networks", len(dc.Networks), metric.GAUGE))
		checkError(config, ms.SetMetric("resourcePools", countResourcePools, metric.GAUGE))
		checkError(config, ms.SetMetric("clusters", len(dc.Clusters), metric.GAUGE))

	}
}
