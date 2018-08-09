package dockerdiscovery

import (
	"testing"
	"net"

	"github.com/mholt/caddy"
	"github.com/stretchr/testify/assert"
	dockerapi "github.com/fsouza/go-dockerclient"

	"log"
)

type setupDockerDiscoveryTestCase struct {
	configBlock            string
	expectedDockerEndpoint string
	expectedDockerDomain   string
}

func TestSetupDockerDiscovery(t *testing.T) {
	testCases := []setupDockerDiscoveryTestCase{
		setupDockerDiscoveryTestCase{
			"docker",
			defaultDockerEndpoint,
			defaultDockerDomain,
		},
		setupDockerDiscoveryTestCase{
			"docker unix:///var/run/docker.sock.backup",
			"unix:///var/run/docker.sock.backup",
			defaultDockerDomain,
		},
		setupDockerDiscoveryTestCase{
			`docker {
	domain example.org.
}`,
			defaultDockerEndpoint,
			"example.org.",
		},
		setupDockerDiscoveryTestCase{
			`docker unix:///home/user/docker.sock {
	domain home.example.org.
}`,
			"unix:///home/user/docker.sock",
			"home.example.org.",
		},
	}

	for _, tc := range testCases {
		c := caddy.NewTestController("dns", tc.configBlock)
		dd, err := createPlugin(c)
		assert.Nil(t, err)
		assert.Equal(t, dd.dockerEndpoint, tc.expectedDockerEndpoint)
	}

	c := caddy.NewTestController("dns", `docker unix:///home/user/docker.sock {
	domain home.example.org.
	network_aliases my_project_network_name
}`)
	dd, err := createPlugin(c)
	assert.Nil(t, err)

	networks := make(map[string]dockerapi.ContainerNetwork)
	var aliases = []string{"myproject.loc"}

	networks["my_project_network_name"] = dockerapi.ContainerNetwork{
		Aliases: aliases,
	}
	var address = net.ParseIP("192.11.0.1")
	var container = &dockerapi.Container{
		ID: "container-1",
		Config: &dockerapi.Config{
			Hostname: "nginx",
		},
		NetworkSettings: &dockerapi.NetworkSettings{
			Networks: networks,
			IPAddress: address.String(),
		},
	}


	e := dd.addContainer(container)
	assert.Nil(t, e)

	containerInfo, e := dd.containerInfoByDomain("myproject.loc.")
	assert.Nil(t, e)
	assert.NotNil(t, containerInfo)
	assert.NotNil(t, containerInfo.address)

	log.Printf("%s", containerInfo.address.Equal(address))
	assert.Equal(t, containerInfo.address, address)

	containerInfo, e = dd.containerInfoByDomain("wrong.loc.")
	assert.Nil(t, containerInfo)

	containerInfo, e = dd.containerInfoByDomain("nginx.home.example.org.")
	assert.NotNil(t, containerInfo)

	containerInfo, e = dd.containerInfoByDomain("wrong.home.example.org.")
	assert.Nil(t, containerInfo)
}
