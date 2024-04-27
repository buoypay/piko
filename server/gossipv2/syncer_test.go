package gossip

import (
	"testing"

	"github.com/andydunstall/pico/pkg/log"
	netmap "github.com/andydunstall/pico/server/netmapv2"
	"github.com/stretchr/testify/assert"
)

type upsert struct {
	Key   string
	Value string
}

type fakeGossiper struct {
	upserts []upsert
	deletes []string
}

func (g *fakeGossiper) UpsertLocal(key, value string) {
	g.upserts = append(g.upserts, upsert{
		Key:   key,
		Value: value,
	})
}

func (g *fakeGossiper) DeleteLocal(key string) {
	g.deletes = append(g.deletes, key)
}

var _ gossiper = &fakeGossiper{}

func TestSyncer_Sync(t *testing.T) {
	localNode := &netmap.Node{
		ID:        "local",
		ProxyAddr: "10.26.104.56:8000",
		AdminAddr: "10.26.104.56:8001",
	}
	m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())
	m.AddLocalEndpoint("my-endpoint")
	m.AddLocalEndpoint("my-endpoint")
	m.AddLocalEndpoint("my-endpoint")

	sync := newSyncer(m, log.NewNopLogger())

	gossiper := &fakeGossiper{}
	sync.Sync(gossiper)

	assert.Equal(
		t,
		[]upsert{
			{"proxy_addr", "10.26.104.56:8000"},
			{"admin_addr", "10.26.104.56:8001"},
			{"endpoint:my-endpoint", "3"},
		},
		gossiper.upserts,
	)
}

func TestSyncer_OnLocalEndpointUpdate(t *testing.T) {
	localNode := &netmap.Node{
		ID:        "local",
		ProxyAddr: "10.26.104.56:8000",
		AdminAddr: "10.26.104.56:8001",
	}
	m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

	sync := newSyncer(m, log.NewNopLogger())

	gossiper := &fakeGossiper{}
	sync.Sync(gossiper)

	m.AddLocalEndpoint("my-endpoint")
	assert.Equal(
		t,
		upsert{"endpoint:my-endpoint", "1"},
		gossiper.upserts[len(gossiper.upserts)-1],
	)

	m.AddLocalEndpoint("my-endpoint")
	assert.Equal(
		t,
		upsert{"endpoint:my-endpoint", "2"},
		gossiper.upserts[len(gossiper.upserts)-1],
	)

	m.RemoveLocalEndpoint("my-endpoint")
	assert.Equal(
		t,
		upsert{"endpoint:my-endpoint", "1"},
		gossiper.upserts[len(gossiper.upserts)-1],
	)

	m.RemoveLocalEndpoint("my-endpoint")
	assert.Equal(
		t,
		"endpoint:my-endpoint",
		gossiper.deletes[len(gossiper.deletes)-1],
	)
}

func TestSyncer_RemoteNodeUpdate(t *testing.T) {
	t.Run("add node", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		sync.OnJoin("remote")
		sync.OnUpsertKey("remote", "proxy_addr", "10.26.104.98:8000")
		sync.OnUpsertKey("remote", "admin_addr", "10.26.104.98:8001")
		sync.OnUpsertKey("remote", "endpoint:my-endpoint", "5")

		node, ok := m.Node("remote")
		assert.True(t, ok)
		assert.Equal(t, node, &netmap.Node{
			ID:        "remote",
			Status:    netmap.NodeStatusActive,
			ProxyAddr: "10.26.104.98:8000",
			AdminAddr: "10.26.104.98:8001",
			Endpoints: map[string]int{
				"my-endpoint": 5,
			},
		})
	})

	t.Run("add node missing state", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		sync.OnJoin("remote")
		sync.OnUpsertKey("remote", "proxy_addr", "10.26.104.98:8000")

		// We don't have the node status therefore it is still pending.
		_, ok := m.Node("remote")
		assert.False(t, ok)
	})

	t.Run("add local node", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			Status:    netmap.NodeStatusActive,
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		// Updates to the local node should have no affect.
		sync.OnJoin("local")
		sync.OnUpsertKey("local", "proxy_addr", "10.26.104.98:8000")
		sync.OnUpsertKey("local", "admin_addr", "10.26.104.98:8001")

		assert.Equal(t, localNode, m.LocalNode())
	})

	t.Run("update node", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			Status:    netmap.NodeStatusActive,
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		sync.OnJoin("remote")
		sync.OnUpsertKey("remote", "proxy_addr", "10.26.104.98:8000")
		sync.OnUpsertKey("remote", "admin_addr", "10.26.104.98:8001")
		sync.OnUpsertKey("remote", "endpoint:my-endpoint", "5")

		_, ok := m.Node("remote")
		assert.True(t, ok)

		sync.OnUpsertKey("remote", "endpoint:my-endpoint-2", "8")
		sync.OnDeleteKey("remote", "endpoint:my-endpoint")

		node, ok := m.Node("remote")
		assert.True(t, ok)
		assert.Equal(t, node, &netmap.Node{
			ID:        "remote",
			Status:    netmap.NodeStatusActive,
			ProxyAddr: "10.26.104.98:8000",
			AdminAddr: "10.26.104.98:8001",
			Endpoints: map[string]int{
				"my-endpoint-2": 8,
			},
		})
	})
}

func TestSyncer_RemoteNodeLeave(t *testing.T) {
	t.Run("active node leave", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		// Add remote node.
		sync.OnJoin("remote")
		sync.OnUpsertKey("remote", "proxy_addr", "10.26.104.98:8000")
		sync.OnUpsertKey("remote", "admin_addr", "10.26.104.98:8001")
		sync.OnUpsertKey("remote", "endpoint:my-endpoint", "5")

		// Leaving should update the netmap.
		sync.OnLeave("remote")

		node, ok := m.Node("remote")
		assert.True(t, ok)
		assert.Equal(t, node, &netmap.Node{
			ID:        "remote",
			Status:    netmap.NodeStatusLeft,
			ProxyAddr: "10.26.104.98:8000",
			AdminAddr: "10.26.104.98:8001",
			Endpoints: map[string]int{
				"my-endpoint": 5,
			},
		})

		sync.OnExpired("remote")

		// Expiring should remove from the netmap.
		_, ok = m.Node("remote")
		assert.False(t, ok)
	})

	t.Run("pending node leave", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		// Add remote node.
		sync.OnJoin("remote")
		sync.OnUpsertKey("remote", "proxy_addr", "10.26.104.98:8000")

		// Leaving should discard the pending node.
		sync.OnLeave("remote")

		sync.OnUpsertKey("remote", "admin_addr", "10.26.104.98:8001")

		_, ok := m.Node("remote")
		assert.False(t, ok)
	})

	t.Run("local node leave", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			Status:    netmap.NodeStatusActive,
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		// Attempting to mark the local node as left should have no affect.
		sync.OnLeave("local")

		assert.Equal(t, localNode, m.LocalNode())
	})
}

func TestSyncer_RemoteNodeDown(t *testing.T) {
	t.Run("active node", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		// Add remote node.
		sync.OnJoin("remote")
		sync.OnUpsertKey("remote", "proxy_addr", "10.26.104.98:8000")
		sync.OnUpsertKey("remote", "admin_addr", "10.26.104.98:8001")
		sync.OnUpsertKey("remote", "endpoint:my-endpoint", "5")

		// Marking a node down should update the netmap.
		sync.OnDown("remote")

		node, ok := m.Node("remote")
		assert.True(t, ok)
		assert.Equal(t, node, &netmap.Node{
			ID:        "remote",
			Status:    netmap.NodeStatusDown,
			ProxyAddr: "10.26.104.98:8000",
			AdminAddr: "10.26.104.98:8001",
			Endpoints: map[string]int{
				"my-endpoint": 5,
			},
		})

		sync.OnExpired("remote")

		// Expiring should remove from the netmap.
		_, ok = m.Node("remote")
		assert.False(t, ok)
	})

	t.Run("active node recovers", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		// Add remote node.
		sync.OnJoin("remote")
		sync.OnUpsertKey("remote", "proxy_addr", "10.26.104.98:8000")
		sync.OnUpsertKey("remote", "admin_addr", "10.26.104.98:8001")
		sync.OnUpsertKey("remote", "endpoint:my-endpoint", "5")

		// Marking a node down should update the netmap.
		sync.OnDown("remote")

		// Marking a node healthy should update the netmap.
		sync.OnHealthy("remote")

		node, ok := m.Node("remote")
		assert.True(t, ok)
		assert.Equal(t, node, &netmap.Node{
			ID:        "remote",
			Status:    netmap.NodeStatusActive,
			ProxyAddr: "10.26.104.98:8000",
			AdminAddr: "10.26.104.98:8001",
			Endpoints: map[string]int{
				"my-endpoint": 5,
			},
		})
	})

	t.Run("pending node down", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		// Add remote node.
		sync.OnJoin("remote")
		sync.OnUpsertKey("remote", "proxy_addr", "10.26.104.98:8000")

		// Marking down should not remove the pending node.
		sync.OnDown("remote")
		sync.OnHealthy("remote")

		sync.OnUpsertKey("remote", "admin_addr", "10.26.104.98:8001")

		node, ok := m.Node("remote")
		assert.True(t, ok)
		assert.Equal(t, node, &netmap.Node{
			ID:        "remote",
			Status:    netmap.NodeStatusActive,
			ProxyAddr: "10.26.104.98:8000",
			AdminAddr: "10.26.104.98:8001",
		})
	})

	t.Run("pending node expires", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		// Add remote node.
		sync.OnJoin("remote")
		sync.OnUpsertKey("remote", "proxy_addr", "10.26.104.98:8000")

		// Marking down should not remove the pending node.
		sync.OnDown("remote")
		sync.OnExpired("remote")

		sync.OnUpsertKey("remote", "admin_addr", "10.26.104.98:8001")

		_, ok := m.Node("remote")
		assert.False(t, ok)
	})

	t.Run("local node leave", func(t *testing.T) {
		localNode := &netmap.Node{
			ID:        "local",
			Status:    netmap.NodeStatusActive,
			ProxyAddr: "10.26.104.56:8000",
			AdminAddr: "10.26.104.56:8001",
		}
		m := netmap.NewNetworkMap(localNode.Copy(), log.NewNopLogger())

		sync := newSyncer(m, log.NewNopLogger())

		gossiper := &fakeGossiper{}
		sync.Sync(gossiper)

		// Attempting to mark the local node as down should have no affect.
		sync.OnLeave("local")

		assert.Equal(t, localNode, m.LocalNode())
	})
}
