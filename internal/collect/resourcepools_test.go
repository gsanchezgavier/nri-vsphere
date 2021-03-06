package collect

import (
	"context"
	"github.com/newrelic/nri-vsphere/internal/client"
	"github.com/newrelic/nri-vsphere/internal/config"
	"github.com/newrelic/nri-vsphere/internal/tag"

	logrus "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"testing"
)

func Test_ListResourcePools_WithNonEmptyFilter(t *testing.T) {
	simulator.Run(func(ctx context.Context, vc *vim25.Client) error {
		vmClient, err := client.New(vc.URL().String(), "user", "pass", false)
		assert.NoError(t, err)

		c := rest.NewClient(vc)
		err = c.Login(ctx, simulator.DefaultLogin)
		assert.NoError(t, err)

		m := tags.NewManager(c)
		addRpTag(ctx, m, vc, "region", "eu")
		addRpTag(ctx, m, vc, "env", "test")

		// given
		collector := tag.NewCollector(m, logrus.StandardLogger())
		_ = collector.BuildTagCache()

		cfg := &config.Config{
			Args: config.ArgumentList{
				EnableVsphereTags: true,
			},
			IsVcenterAPIType: true,
			VMWareClient:     vmClient,
			ViewManager:      view.NewManager(vc),
			TagCollector:     collector,
			Logrus:           logrus.StandardLogger(),
		}

		tests := []struct {
			name string
			args string
			want bool
		}{
			{
				name: "ByNonExistingTag",
				args: "key=value",
				want: false,
			},
			{
				name: "ByExistingTag",
				args: "region=eu",
				want: true,
			},
			{
				name: "ByMultipleMixedTags",
				args: "key=value env=test",
				want: true,
			},
			{
				name: "ByMultipleExistingTags",
				args: "region=eu env=test",
				want: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// we need to get the datacenter first
				cfg.Datacenters = append(cfg.Datacenters, getDatacenter(ctx, cfg.ViewManager))
				cfg.Datacenters[0].ResourcePools = make(map[types.ManagedObjectReference]*mo.ResourcePool)
				// make sure tag filtering is enabled
				cfg.Args.IncludeTags = tt.args
				collector.ParseFilterTagExpression(tt.args)

				// when
				ResourcePools(cfg)

				// then
				for k := range cfg.Datacenters[0].ResourcePools {
					actual := collector.MatchObjectTags(k)
					assert.Equal(t, tt.want, actual)
				}
			})
		}

		return nil
	})
}

func addRpTag(ctx context.Context, m *tags.Manager, vc *vim25.Client, category string, value string) {
	categoryID, _ := m.CreateCategory(ctx, &tags.Category{
		AssociableTypes: []string{RESOURCE_POOL},
		Cardinality:     "SINGLE",
		Name:            category,
	})
	tagID, _ := m.CreateTag(ctx, &tags.Tag{CategoryID: categoryID, Name: value})
	finder := find.NewFinder(vc)
	rps, _ := finder.ResourcePoolList(ctx, "/DC0/...")
	for _, rp := range rps {
		_ = m.AttachTag(ctx, tagID, rp.Reference())
	}
}
