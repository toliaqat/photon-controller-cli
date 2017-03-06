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
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/vmware/photon-controller-cli/photon/client"
	"github.com/vmware/photon-controller-cli/photon/utils"

	"github.com/urfave/cli"
	"github.com/vmware/photon-controller-go-sdk/photon"
)

const (
	PHYSICAL         = "PHYSICAL"
	SOFTWARE_DEFINED = "SOFTWARE_DEFINED"
	NOT_AVAILABLE    = "NOT_AVAILABLE"
)

// Creates a cli.Command for networks
// Subcommands: create; Usage: network create [<options>]
//              delete; Usage: network delete <id>
//              list;   Usage: network list
//              show;   Usage: network show <id>
//              set-default; Usage: network setDefault <id>
func GetNetworksCommand() cli.Command {
	command := cli.Command{
		Name:  "network",
		Usage: "options for network",
		Subcommands: []cli.Command{
			{
				Name:      "create",
				Usage:     "Create a new network",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Network name",
					},
					cli.StringFlag{
						Name:  "description, d",
						Usage: "Description of network",
					},
					cli.StringFlag{
						Name:  "portgroups, p",
						Usage: "PortGroups associated with network (only for physical network)",
					},
					cli.StringFlag{
						Name: "routingType, r",
						Usage: "Routing type for network (only for software-defined network). Supported values are: " +
							"'ROUTED' and 'ISOLATED'",
					},
					cli.StringFlag{
						Name:  "size, s",
						Usage: "Size of the private IP addresses (only for software-defined network)",
					},
					cli.StringFlag{
						Name:  "staticIpSize, f",
						Usage: "Size of the reserved static IP addresses (only for software-defined network)",
					},
					cli.StringFlag{
						Name:  "projectId, i",
						Usage: "ID of the project that network belongs to (only for software-defined network)",
					},
				},
				Action: func(c *cli.Context) {
					sdnEnabled, err := isSoftwareDefinedNetwork(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}

					if sdnEnabled {
						err = createVirtualNetwork(c, os.Stdout)
					} else {
						err = createPhysicalNetwork(c, os.Stdout)
					}
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete a network",
				ArgsUsage: "<network-id>",
				Action: func(c *cli.Context) {
					err := deleteNetwork(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:      "list",
				Usage:     "List networks",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Optionally filter by name",
					},
					cli.StringFlag{
						Name: "projectId, i",
						Usage: "ID of the project that networks to be listed belong to (only for software-defined " +
							"network)",
					},
				},
				Action: func(c *cli.Context) {
					sdnEnabled, err := isSoftwareDefinedNetwork(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}

					if sdnEnabled {
						err = listVirtualNetworks(c, os.Stdout)
					} else {
						err = listPhysicalNetworks(c, os.Stdout)
					}
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:      "show",
				Usage:     "Show network given its id",
				ArgsUsage: "<network-id>",
				Action: func(c *cli.Context) {
					sdnEnabled, err := isSoftwareDefinedNetwork(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}

					if sdnEnabled {
						err = showVirtualNetwork(c, os.Stdout)
					} else {
						err = showPhysicalNetwork(c, os.Stdout)
					}
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:      "set-default",
				Usage:     "Set default network",
				ArgsUsage: "<network-id>",
				Description: "Set the default network to be used in the current project when making a VM\n" +
					"   This is not required: when making a VM you can either specify the network to use, or rely\n" +
					"   on the default network.",
				Action: func(c *cli.Context) {
					err := setDefaultNetwork(c, os.Stdout)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
		},
	}
	return command
}

func isSoftwareDefinedNetwork(c *cli.Context) (sdnEnabled bool, err error) {
	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return
	}

	info, err := client.Photonclient.Info.Get()
	if err != nil {
		return
	}

	if info.NetworkType == NOT_AVAILABLE {
		err = errors.New("Network type is missing")
	} else {
		sdnEnabled = (info.NetworkType == SOFTWARE_DEFINED)
	}
	return
}

func deleteNetwork(c *cli.Context) error {
	err := checkArgCount(c, 1)
	if err != nil {
		return err
	}
	id := c.Args().First()

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	sdnEnabled, err := isSoftwareDefinedNetwork(c)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	var task *photon.Task
	if !confirmed(c) {
		fmt.Println("Canceled")
		return nil
	}

	if sdnEnabled {
		task, err = client.Photonclient.VirtualSubnets.Delete(id)
	} else {
		task, err = client.Photonclient.Networks.Delete(id)
	}
	if err != nil {
		return err
	}

	_, err = waitOnTaskOperation(task.ID, c)
	if err != nil {
		return err
	}
	return nil
}

func setDefaultNetwork(c *cli.Context, w io.Writer) error {
	err := checkArgCount(c, 1)
	if err != nil {
		return err
	}
	id := c.Args().First()

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	sdnEnabled, err := isSoftwareDefinedNetwork(c)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	var task *photon.Task
	if sdnEnabled {
		task, err = client.Photonclient.VirtualSubnets.SetDefault(id)
	} else {
		task, err = client.Photonclient.Networks.SetDefault(id)
	}

	if err != nil {
		return err
	}

	if confirmed(c) {
		id, err := waitOnTaskOperation(task.ID, c)
		if err != nil {
			return err
		}

		if utils.NeedsFormatting(c) {
			network, err := client.Photonclient.Networks.Get(id)
			if err != nil {
				return err
			}
			utils.FormatObject(network, w, c)
		}
	} else {
		fmt.Println("OK. Canceled")
	}
	return nil
}
