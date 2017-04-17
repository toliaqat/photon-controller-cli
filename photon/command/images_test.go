// Copyright (c) 2016 VMware, Inc. All Rights Reserved.
//
// This product is licensed to you under the Apache License, Version 2.0 (the "License").
// You may not use this product except in compliance with the License.
//
// This product may include a number of subcomponents with separate copyright notices and
// license terms. Your use of these subcomponents is subject to the terms and conditions
// of the subcomponent's license, as noted in the LICENSE file.

package command

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"testing"

	"github.com/vmware/photon-controller-cli/photon/client"
	"github.com/vmware/photon-controller-cli/photon/mocks"

	"github.com/urfave/cli"
	"github.com/vmware/photon-controller-go-sdk/photon"
)

type MockImagesPage struct {
	Items            []photon.Image `json:"items"`
	NextPageLink     string         `json:"nextPageLink"`
	PreviousPageLink string         `json:"previousPageLink"`
}

func TestCreateDeleteImage(t *testing.T) {
	queuedTask := &photon.Task{
		Operation: "CREATE_IMAGE",
		State:     "QUEUED",
		Entity:    photon.Entity{ID: "1"},
	}
	completedTask := &photon.Task{
		Operation: "CREATE_IMAGE",
		State:     "COMPLETED",
		Entity:    photon.Entity{ID: "1"},
	}

	response, err := json.Marshal(queuedTask)
	if err != nil {
		t.Error("Not expecting error serializaing expected queuedTask")
	}
	taskresponse, err := json.Marshal(completedTask)
	if err != nil {
		t.Error("Not expecting error serializaing expected createTask")
	}

	const projectId = "project1"
	server := mocks.NewTestServer()
	mocks.RegisterResponder(
		"POST",
		server.URL+rootUrl+"/projects/"+projectId+"/images",
		mocks.CreateResponder(200, string(response[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+rootUrl+"/tasks/"+queuedTask.ID,
		mocks.CreateResponder(200, string(taskresponse[:])))
	defer server.Close()

	mocks.Activate(true)
	httpClient := &http.Client{Transport: mocks.DefaultMockTransport}
	client.Photonclient = photon.NewTestClient(server.URL, nil, httpClient)

	set := flag.NewFlagSet("test", 0)
	set.String("name", "n", "testname")
	set.String("image_replication", "EAGER", "image replication")
	set.String("scope", "project", "image scope")
	set.String("project", projectId, "project id")
	err = set.Parse([]string{"../../testdata/tty_tiny.ova"})
	if err != nil {
		t.Error("Not expecting arguments parsing to fail")
	}
	cxt := cli.NewContext(nil, set, nil)

	err = createImage(cxt, os.Stdout)
	if err != nil {
		t.Error("Not expecting error creating image: " + err.Error())
	}

	queuedTask = &photon.Task{
		Operation: "DELETE_IMAGE",
		State:     "QUEUED",
		Entity:    photon.Entity{ID: "1"},
	}
	completedTask = &photon.Task{
		Operation: "DELETE_IMAGE",
		State:     "COMPLETED",
		Entity:    photon.Entity{ID: "1"},
	}

	response, err = json.Marshal(queuedTask)
	if err != nil {
		t.Error("Not expecting error serializaing expected queuedTask")
	}
	taskresponse, err = json.Marshal(completedTask)
	if err != nil {
		t.Error("Not expecting error serializaing expected deleteTask")
	}

	mocks.RegisterResponder(
		"DELETE",
		server.URL+rootUrl+"/images/"+queuedTask.Entity.ID,
		mocks.CreateResponder(200, string(response[:])))
	mocks.RegisterResponder(
		"GET",
		server.URL+rootUrl+"/tasks/"+queuedTask.ID,
		mocks.CreateResponder(200, string(taskresponse[:])))

	globalSet := flag.NewFlagSet("global", 0)
	globalSet.Bool("non-interactive", true, "doc")
	err = globalSet.Parse([]string{"--non-interactive"})
	if err != nil {
		t.Error("Not expecting arguments parsing to fail")
	}
	globalCtx := cli.NewContext(nil, globalSet, nil)
	set = flag.NewFlagSet("test", 0)
	err = set.Parse([]string{queuedTask.Entity.ID})
	if err != nil {
		t.Error("Not expecting arguments parsing to fail")
	}

	cxt = cli.NewContext(nil, set, globalCtx)
	err = deleteImage(cxt)
	if err != nil {
		t.Error("Not expecting error deleting image: " + err.Error())
	}
}

func TestFindImagesByName(t *testing.T) {
	expectedImageList := MockImagesPage{
		Items: []photon.Image{
			{
				Name:            "test",
				Size:            10,
				State:           "COMPLETED",
				ID:              "1",
				ReplicationType: "EAGER",
				Settings: []photon.ImageSetting{
					{
						Name:         "test-setting",
						DefaultValue: "test-default-value",
					},
				},
			},
			{
				Name:            "test2",
				Size:            10,
				State:           "COMPLETED",
				ID:              "2",
				ReplicationType: "EAGER",
				Settings: []photon.ImageSetting{
					{
						Name:         "test-setting",
						DefaultValue: "test-default-value",
					},
				},
			},
		},
		NextPageLink:     "fake-next-page-link",
		PreviousPageLink: "",
	}

	response, err := json.Marshal(expectedImageList)
	if err != nil {
		t.Error("Not expecting error serializaing expected response")
	}

	server := mocks.NewTestServer()
	mocks.RegisterResponder(
		"GET",
		server.URL+rootUrl+"/images?name=test",
		mocks.CreateResponder(200, string(response[:])))

	expectedImageList = MockImagesPage{
		Items:            []photon.Image{},
		NextPageLink:     "",
		PreviousPageLink: "",
	}

	response, err = json.Marshal(expectedImageList)
	if err != nil {
		t.Error("Not expecting error serializaing expected response")
	}

	mocks.RegisterResponder(
		"GET",
		server.URL+"fake-next-page-link",
		mocks.CreateResponder(200, string(response[:])))

	defer server.Close()

	mocks.Activate(true)
	httpClient := &http.Client{Transport: mocks.DefaultMockTransport}
	client.Photonclient = photon.NewTestClient(server.URL, nil, httpClient)

	set := flag.NewFlagSet("test", 0)
	set.String("name", "test", "image name")
	if err != nil {
		t.Error("Not expecting arguments parsing to fail")
	}
	cxt := cli.NewContext(nil, set, nil)
	err = listImages(cxt, os.Stdout)
	if err != nil {
		t.Error("Not expecting an error listing images ", err)
	}
}

func TestListImage(t *testing.T) {
	expectedImageList := MockImagesPage{
		Items: []photon.Image{
			{
				Name:            "test",
				Size:            10,
				State:           "COMPLETED",
				ID:              "1",
				ReplicationType: "EAGER",
				Settings: []photon.ImageSetting{
					{
						Name:         "test-setting",
						DefaultValue: "test-default-value",
					},
				},
			},
		},
		NextPageLink:     "fake-next-page-link",
		PreviousPageLink: "",
	}

	response, err := json.Marshal(expectedImageList)
	if err != nil {
		t.Error("Not expecting error serializaing expected response")
	}

	server := mocks.NewTestServer()
	mocks.RegisterResponder(
		"GET",
		server.URL+rootUrl+"/images",
		mocks.CreateResponder(200, string(response[:])))

	expectedImageList = MockImagesPage{
		Items:            []photon.Image{},
		NextPageLink:     "",
		PreviousPageLink: "",
	}

	response, err = json.Marshal(expectedImageList)
	if err != nil {
		t.Error("Not expecting error serializaing expected response")
	}

	mocks.RegisterResponder(
		"GET",
		server.URL+"fake-next-page-link",
		mocks.CreateResponder(200, string(response[:])))

	defer server.Close()

	mocks.Activate(true)
	httpClient := &http.Client{Transport: mocks.DefaultMockTransport}
	client.Photonclient = photon.NewTestClient(server.URL, nil, httpClient)

	globalFlags := flag.NewFlagSet("global-flags", flag.ContinueOnError)
	globalFlags.String("output", "json", "output")
	err = globalFlags.Parse([]string{"--output=json"})
	if err != nil {
		t.Error(err)
	}
	globalCtx := cli.NewContext(nil, globalFlags, nil)
	commandFlags := flag.NewFlagSet("command-flags", flag.ContinueOnError)
	err = commandFlags.Parse([]string{})
	if err != nil {
		t.Error(err)
	}
	cxt := cli.NewContext(nil, commandFlags, globalCtx)
	err = listImages(cxt, os.Stdout)
	if err != nil {
		t.Errorf("Not expecting an error listing images: %s", err)
	}

	// Now validate the output
	var output bytes.Buffer
	err = listImages(cxt, &output)

	// Verify it didn't fail
	if err != nil {
		t.Errorf("List images failed: %s", err)
	}

	// Verify we printed a list of images: it should start with a bracket
	err = checkRegExp(`^\s*\[`, output)
	if err != nil {
		t.Errorf("List images didn't produce a JSON list that starts with a bracket (list): %s", err)
	}
	// and end with a bracket (two regular expressions because it's multiline, it's easier)
	err = checkRegExp(`\]\s*$`, output)
	if err != nil {
		t.Errorf("List images didn't produce JSON that ended in a bracket (list): %s", err)
	}
	// And spot check that we have the "id" field
	err = checkRegExp(`\"id\":\s*\".*\"`, output)
	if err != nil {
		t.Errorf("List images didn't produce a JSON field named 'id': %s", err)
	}
}

func TestImageTasks(t *testing.T) {
	taskList := MockTasksPage{
		Items: []photon.Task{
			{
				Operation: "CREATE_IMAGE",
				State:     "COMPLETED",
				ID:        "1",
				Entity:    photon.Entity{ID: "1", Kind: "image"},
			},
		},
		NextPageLink:     "/fake-next-page-link",
		PreviousPageLink: "",
	}

	response, err := json.Marshal(taskList)
	if err != nil {
		t.Error("Not expecting error serializaing expected status")
	}

	server := mocks.NewTestServer()
	mocks.RegisterResponder(
		"GET",
		server.URL+rootUrl+"/images/1/tasks",
		mocks.CreateResponder(200, string(response[:])))

	taskList = MockTasksPage{
		Items:            []photon.Task{},
		NextPageLink:     "",
		PreviousPageLink: "",
	}
	response, err = json.Marshal(taskList)
	if err != nil {
		t.Error("Not expecting error serializing expected taskLists")
	}

	mocks.RegisterResponder(
		"GET",
		server.URL+"/fake-next-page-link",
		mocks.CreateResponder(200, string(response[:])))

	defer server.Close()

	mocks.Activate(true)
	httpClient := &http.Client{Transport: mocks.DefaultMockTransport}
	client.Photonclient = photon.NewTestClient(server.URL, nil, httpClient)

	set := flag.NewFlagSet("test", 0)
	err = set.Parse([]string{"1"})
	if err != nil {
		t.Error("Not expecting arguments parsing to fail")
	}
	cxt := cli.NewContext(nil, set, nil)
	err = getImageTasks(cxt)
	if err != nil {
		t.Error("Not expecting error retrieving tenant tasks")
	}
}
