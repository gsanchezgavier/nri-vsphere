// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collect

import (
	"context"
	"github.com/newrelic/nri-vsphere/internal/config"
	"github.com/newrelic/nri-vsphere/internal/performance"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Hosts VMWare
func Hosts(config *config.Config) {

	ctx := context.Background()
	m := config.ViewManager

	// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.HostSystem.html
	propertiesToRetrieve := []string{"summary", "overallStatus", "config", "network", "vm", "runtime", "parent", "datastore"}
	for i, dc := range config.Datacenters {
		logger := config.Logrus.WithField("datacenter", dc.Datacenter.Name)

		cv, err := m.CreateContainerView(ctx, dc.Datacenter.Reference(), []string{HOST}, true)
		if err != nil {
			logger.WithError(err).Error("failed to create HostSystem container view")
			continue
		}

		defer func() {
			err := cv.Destroy(ctx)
			if err != nil {
				logger.WithError(err).Error("error while cleaning up host container view")
			}
		}()

		var hosts []mo.HostSystem
		err = cv.Retrieve(ctx, []string{HOST}, propertiesToRetrieve, &hosts)
		if err != nil {
			logger.WithError(err).Error("failed to retrieve HostSystems")
			continue
		}

		if config.TagCollectionEnabled() {
			_, err = config.TagCollector.FetchTagsForObjects(hosts)
			if err != nil {
				logger.WithError(err).Warn("failed to retrieve tags for hosts", err)
			} else {
				logger.WithField("seconds", config.Uptime()).Debug("hosts tags collected")
			}
		}

		var hostsRefs []types.ManagedObjectReference
		for j, host := range hosts {
			config.Datacenters[i].Hosts[host.Self] = &hosts[j]

			// filtering here only affects performance metrics collection
			if config.TagFilteringEnabled() && !config.TagCollector.MatchObjectTags(host.Reference()) {
				continue
			}
			hostsRefs = append(hostsRefs, host.Self)
		}

		if config.PerfMetricsCollectionEnabled() {
			metricsToCollect := config.PerfCollector.MetricDefinition.Host
			collectedData := config.PerfCollector.Collect(hostsRefs, metricsToCollect, performance.RealTimeInterval)
			dc.AddPerfMetrics(collectedData)

			logger.WithField("seconds", config.Uptime()).Debug("hosts perf metrics collected")
		}
	}
}
