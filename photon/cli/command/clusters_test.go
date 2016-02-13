package command

import (
	"encoding/json"
	"flag"
	"net/http"
	"testing"

	"github.com/vmware/photon-controller-cli/photon/cli/client"
	"github.com/vmware/photon-controller-cli/photon/cli/mocks"

	"github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/vmware/photon-controller-go-sdk/photon"
)

type MockClustersPage struct {
	Items            []photon.Cluster `json:"items"`
	NextPageLink     string           `json:"nextPageLink"`
	PreviousPageLink string           `json:"previousPageLink"`
}

func TestCreateDeleteCluster(t *testing.T) {
	tenantStruct := photon.Tenants{
		Items: []photon.Tenant{
			photon.Tenant{
				Name: "fake_tenant_name",
				ID:   "fake_tenant_id",
			},
		},
	}
	tenantResponse, err := json.Marshal(tenantStruct)
	if err != nil {
		t.Error("Not expecting error serializing expected tenantStruct")
	}

	projectStruct := photon.ProjectList{
		Items: []photon.ProjectCompact{
			photon.ProjectCompact{
				Name: "fake_project_name",
				ID:   "fake_project_id",
			},
		},
	}
	projectResponse, err := json.Marshal(projectStruct)
	if err != nil {
		t.Error("Not expecting error serializing expected projectStruct")
	}

	queuedCreationTask := &photon.Task{
		Operation: "CREATE_CLUSTER",
		State:     "QUEUED",
		ID:        "fake_create_cluster_task_id",
		Entity:    photon.Entity{ID: "fake_cluster_id"},
	}
	queuedCreationTaskResponse, err := json.Marshal(queuedCreationTask)
	if err != nil {
		t.Error("Not expecting error serializing expected queued creation task")
	}

	completedCreationTask := &photon.Task{
		Operation: "CREATE_CLUSTER",
		State:     "COMPLETED",
		ID:        "fake_create_cluster_task_id",
		Entity:    photon.Entity{ID: "fake_cluster_id"},
	}
	completedCreationTaskResponse, err := json.Marshal(completedCreationTask)
	if err != nil {
		t.Error("Not expecting error serializing expected completed creation task")
	}

	server = mocks.NewTestServer()
	defer server.Close()

	mocks.RegisterResponder(
		"GET",
		server.URL+"/tenants",
		mocks.CreateResponder(200, string(tenantResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/tenants/fake_tenant_id/projects?name=fake_project_name",
		mocks.CreateResponder(200, string(projectResponse[:])))
	mocks.RegisterResponder(
		"POST",
		server.URL+"/projects/fake_project_id/clusters",
		mocks.CreateResponder(200, string(queuedCreationTaskResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/tasks/"+queuedCreationTask.ID,
		mocks.CreateResponder(200, string(completedCreationTaskResponse[:])))
	mocks.Activate(true)

	httpClient := &http.Client{Transport: mocks.DefaultMockTransport}
	client.Esxclient = photon.NewTestClient(server.URL, "", nil, httpClient)

	globalSet := flag.NewFlagSet("test", 0)
	globalSet.Bool("non-interactive", true, "doc")
	globalCtx := cli.NewContext(nil, globalSet, nil)
	err = globalSet.Parse([]string{"--non-interactive"})
	if err != nil {
		t.Error("Not expecting argument parsing to fail")
	}

	set := flag.NewFlagSet("test", 0)
	set.String("tenant", "fake_tenant_name", "tenant name")
	set.String("project", "fake_project_name", "project name")
	set.String("name", "fake_cluster_name", "cluster name")
	set.String("type", "KUBERNETES", "cluster type")
	set.String("vm_flavor", "fake_vm_flavor", "vm flavor name")
	set.String("disk_flavor", "fake_disk_flavor", "disk flavor name")
	set.Int("slave_count", 50, "slave count")
	set.String("dns", "1.1.1.1", "VM network DNS")
	set.String("gateway", "1.1.1.2", "VM network gateway")
	set.String("netmask", "0.0.0.255", "VM network netmask")
	ctx := cli.NewContext(nil, set, globalCtx)

	err = createCluster(ctx)
	if err != nil {
		t.Error("Not expecting error creating cluster: " + err.Error())
	}

	queuedDeletionTask := &photon.Task{
		Operation: "DELETE_CLUSTER",
		State:     "QUEUED",
		ID:        "fake_delete_cluster_task_id",
		Entity:    photon.Entity{ID: "fake_cluster_id"},
	}
	queuedDeletionTaskResponse, err := json.Marshal(queuedDeletionTask)
	if err != nil {
		t.Error("Not expecting error serializing expected queued deletion task")
	}

	completedDeletionTask := &photon.Task{
		Operation: "DELETE_CLUSTER",
		State:     "COMPLETED",
		ID:        "fake_delete_cluster_task_id",
		Entity:    photon.Entity{ID: "fake_cluster_id"},
	}
	completedDeletionTaskResponse, err := json.Marshal(completedDeletionTask)
	if err != nil {
		t.Error("Not expecting error serializing expected completed deletion task")
	}

	mocks.RegisterResponder(
		"DELETE",
		server.URL+"/clusters/fake_cluster_id",
		mocks.CreateResponder(200, string(queuedDeletionTaskResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/tasks/"+queuedDeletionTask.ID,
		mocks.CreateResponder(200, string(completedDeletionTaskResponse[:])))

	set = flag.NewFlagSet("test", 0)
	err = set.Parse([]string{"fake_cluster_id"})
	if err != nil {
		t.Error("Not expecting argument parsing to fail")
	}

	ctx = cli.NewContext(nil, set, globalCtx)
	err = deleteCluster(ctx)
	if err != nil {
		t.Error("Not expecting error deleting cluster: " + err.Error())
	}
}

func TestShowCluster(t *testing.T) {
	cluster := &photon.Cluster{
		Name:       "fake_cluster_name",
		State:      "ERROR",
		ID:         "fake_cluster_id",
		Type:       "KUBERNETES",
		SlaveCount: 50,
	}
	clusterResponse, err := json.Marshal(cluster)
	if err != nil {
		t.Error("Not expecting error serializing expected cluster")
	}

	vmListStruct := photon.VMs{
		Items: []photon.VM{
			photon.VM{
				Name:          "fake_vm_name",
				ID:            "fake_vm_id",
				Flavor:        "fake_vm_flavor_name",
				State:         "STOPPED",
				SourceImageID: "fake_image_id",
				Host:          "fake_host_ip",
				Datastore:     "fake_datastore_ID",
				Tags: []string{
					"cluster:" + cluster.ID + ":master",
				},
				AttachedDisks: []photon.AttachedDisk{
					photon.AttachedDisk{
						Name:       "d1",
						Kind:       "ephemeral-disk",
						Flavor:     "fake_ephemeral_flavor_id",
						CapacityGB: 0,
						BootDisk:   true,
					},
				},
			},
		},
	}
	vmListResponse, err := json.Marshal(vmListStruct)
	if err != nil {
		t.Error("Not expecting error serializing expected vmList")
	}

	queuedNetworkTask := &photon.Task{
		Operation: "GET_NETWORKS",
		State:     "COMPLETED",
		ID:        "fake_get_networks_task_id",
		Entity:    photon.Entity{ID: "fake_vm_id"},
	}
	queuedNetworkTaskResponse, err := json.Marshal(queuedNetworkTask)
	if err != nil {
		t.Error("Not expecting error serializing expected queuedNetworkTask")
	}

	networkMap := make(map[string]interface{})
	networkMap["network"] = "VMmgmtNetwork"
	networkMap["macAddress"] = "00:0c:29:7a:b4:d5"
	networkMap["ipAddress"] = "10.144.121.12"
	networkMap["netmask"] = "255.255.252.0"
	networkMap["isConnected"] = "true"
	networkConnectionMap := make(map[string]interface{})
	networkConnectionMap["networkConnections"] = []interface{}{networkMap}

	completedNetworkTask := &photon.Task{
		Operation:          "GET_NETWORKS",
		State:              "COMPLETED",
		ResourceProperties: networkConnectionMap,
	}
	completedNetworkTaskResponse, err := json.Marshal(completedNetworkTask)
	if err != nil {
		t.Error("Not expecting error serializing expected completedNetworkTask")
	}

	server = mocks.NewTestServer()
	defer server.Close()

	mocks.RegisterResponder(
		"GET",
		server.URL+"/clusters/"+cluster.ID,
		mocks.CreateResponder(200, string(clusterResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/clusters/"+cluster.ID+"/vms",
		mocks.CreateResponder(200, string(vmListResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/vms/"+"fake_vm_id"+"/networks",
		mocks.CreateResponder(200, string(queuedNetworkTaskResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/tasks/"+queuedNetworkTask.ID,
		mocks.CreateResponder(200, string(completedNetworkTaskResponse[:])))
	mocks.Activate(true)

	httpClient := &http.Client{Transport: mocks.DefaultMockTransport}
	client.Esxclient = photon.NewTestClient(server.URL, "", nil, httpClient)

	set := flag.NewFlagSet("test", 0)
	err = set.Parse([]string{"fake_cluster_id"})
	if err != nil {
		t.Error("Not expecting argument parsing to fail")
	}
	ctx := cli.NewContext(nil, set, nil)

	err = showCluster(ctx)
	if err != nil {
		t.Error("Not expecting error showing cluster: " + err.Error())
	}
}

func TestListClusters(t *testing.T) {
	tenantStruct := photon.Tenants{
		Items: []photon.Tenant{
			photon.Tenant{
				Name: "fake_tenant_name",
				ID:   "fake_tenant_id",
			},
		},
	}
	tenantResponse, err := json.Marshal(tenantStruct)
	if err != nil {
		t.Error("Not expecting error serializing expected tenantStruct")
	}

	projectStruct := photon.ProjectList{
		Items: []photon.ProjectCompact{
			photon.ProjectCompact{
				Name: "fake_project_name",
				ID:   "fake_project_id",
			},
		},
	}
	projectResponse, err := json.Marshal(projectStruct)
	if err != nil {
		t.Error("Not expecting error serializing expected projectStruct")
	}

	firstClustersPage := MockClustersPage{
		Items: []photon.Cluster{
			photon.Cluster{
				Name:       "fake_cluster_name",
				State:      "READY",
				ID:         "fake_cluster_id",
				Type:       "KUBERNETES",
				SlaveCount: 50,
			},
		},
		NextPageLink:     "/fake-next-page-link",
		PreviousPageLink: "",
	}

	firstClustersPageResponse, err := json.Marshal(firstClustersPage)
	if err != nil {
		t.Error("Not expecting error serializing expected first clusters page")
	}

	secondClustersPage := MockClustersPage{
		Items:            []photon.Cluster{},
		NextPageLink:     "",
		PreviousPageLink: "",
	}

	secondClustersPageResponse, err := json.Marshal(secondClustersPage)
	if err != nil {
		t.Error("Not expecting error serializing expected second clusters page")
	}

	server := mocks.NewTestServer()
	defer server.Close()

	mocks.RegisterResponder(
		"GET",
		server.URL+"/tenants",
		mocks.CreateResponder(200, string(tenantResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/tenants/fake_tenant_id/projects?name=fake_project_name",
		mocks.CreateResponder(200, string(projectResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/projects/fake_project_id/clusters",
		mocks.CreateResponder(200, string(firstClustersPageResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/fake-next-page-link",
		mocks.CreateResponder(200, string(secondClustersPageResponse[:])))
	mocks.Activate(true)

	httpClient := &http.Client{Transport: mocks.DefaultMockTransport}
	client.Esxclient = photon.NewTestClient(server.URL, "", nil, httpClient)

	set := flag.NewFlagSet("test", 0)
	set.String("tenant", "fake_tenant_name", "tenant name")
	set.String("project", "fake_project_name", "project name")
	ctx := cli.NewContext(nil, set, nil)

	err = listClusters(ctx)
	if err != nil {
		t.Error("Not expecting error listing clusters: " + err.Error())
	}
}

func TestResizeCluster(t *testing.T) {
	queuedTask := &photon.Task{
		Operation: "RESIZE_CLUSTER",
		State:     "QUEUED",
		ID:        "fake_resize_cluster_task_id",
		Entity:    photon.Entity{ID: "fake_cluster_id"},
	}
	queuedTaskResponse, err := json.Marshal(queuedTask)
	if err != nil {
		t.Error("Not expecting error serializing expected queued task")
	}

	completedTask := &photon.Task{
		Operation: "RESIZE_CLUSTER",
		State:     "COMPLETED",
		ID:        "fake_resize_cluster_task_id",
		Entity:    photon.Entity{ID: "fake_cluster_id"},
	}
	completedTaskResponse, err := json.Marshal(completedTask)
	if err != nil {
		t.Error("Not expecting error serializing expected completed task")
	}

	server := mocks.NewTestServer()
	defer server.Close()

	mocks.RegisterResponder(
		"POST",
		server.URL+"/clusters/fake_cluster_id/resize",
		mocks.CreateResponder(200, string(queuedTaskResponse[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+"/tasks/fake_resize_cluster_task_id",
		mocks.CreateResponder(200, string(completedTaskResponse[:])))
	mocks.Activate(true)

	httpClient := &http.Client{Transport: mocks.DefaultMockTransport}
	client.Esxclient = photon.NewTestClient(server.URL, "", nil, httpClient)

	globalSet := flag.NewFlagSet("test", 0)
	globalSet.Bool("non-interactive", true, "doc")
	globalCtx := cli.NewContext(nil, globalSet, nil)
	err = globalSet.Parse([]string{"--non-interactive"})
	if err != nil {
		t.Error("Not expecting argument parsing to fail")
	}

	set := flag.NewFlagSet("test", 0)
	err = set.Parse([]string{"fake_cluster_id", "50"})
	if err != nil {
		t.Error("Not expecting argument parsing to fail")
	}
	ctx := cli.NewContext(nil, set, globalCtx)

	err = resizeCluster(ctx)
	if err != nil {
		t.Error("Not expecting error resizing cluster: " + err.Error())
	}
}

func TestListClusterVms(t *testing.T) {
	server := mocks.NewTestServer()
	defer server.Close()

	vmList := MockVMsPage{
		Items: []photon.VM{
			photon.VM{
				Name:          "fake_vm_name",
				ID:            "fake_vm_ID",
				Flavor:        "fake_vm_flavor_name",
				State:         "STOPPED",
				SourceImageID: "fake_image_ID",
				Host:          "fake_host_ip",
				Datastore:     "fake_datastore_ID",
				AttachedDisks: []photon.AttachedDisk{
					photon.AttachedDisk{
						Name:       "d1",
						Kind:       "ephemeral-disk",
						Flavor:     "fake_ephemeral_flavor_ID",
						CapacityGB: 0,
						BootDisk:   true,
					},
				},
			},
		},
		NextPageLink:     "/fake-next-page-link",
		PreviousPageLink: "",
	}

	listResponse, err := json.Marshal(vmList)
	if err != nil {
		t.Error("Not expecting error serializaing expected vmList")
	}

	mocks.RegisterResponder(
		"GET",
		server.URL+"/clusters/fake_cluster_id/vms",
		mocks.CreateResponder(200, string(listResponse[:])))

	vmList = MockVMsPage{
		Items:            []photon.VM{},
		NextPageLink:     "",
		PreviousPageLink: "",
	}

	listResponse, err = json.Marshal(vmList)
	if err != nil {
		t.Error("Not expecting error serializaing expected vmList")
	}

	mocks.RegisterResponder(
		"GET",
		server.URL+"/fake-next-page-link",
		mocks.CreateResponder(200, string(listResponse[:])))

	mocks.Activate(true)
	httpClient := &http.Client{Transport: mocks.DefaultMockTransport}
	client.Esxclient = photon.NewTestClient(server.URL, "", nil, httpClient)

	set := flag.NewFlagSet("test", 0)
	err = set.Parse([]string{"fake_cluster_id"})
	if err != nil {
		t.Error("Not expecting argument parsing to fail")
	}
	ctx := cli.NewContext(nil, set, nil)

	err = listVms(ctx)
	if err != nil {
		t.Error("Not expecting error listing cluster VMs: " + err.Error())
	}
}
