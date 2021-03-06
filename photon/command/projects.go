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
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/vmware/photon-controller-cli/photon/client"
	cf "github.com/vmware/photon-controller-cli/photon/configuration"
	"github.com/vmware/photon-controller-cli/photon/utils"

	"github.com/urfave/cli"
	"github.com/vmware/photon-controller-go-sdk/photon"
)

// Creates a cli.Command for project
// Subcommands: create; Usage: project create <name> [<options>]
//              delete; Usage: project delete <id>
//              show;   Usage: project show <id>
//              set;    Usage: project set <name>
//              get;    Usage: project get
//              list;   Usage: project list [<options>]
//              tasks;  Usage: project tasks <id> [<options>]
//              quota;  Usage: project quota <operation> <name> [<options>]
func GetProjectsCommand() cli.Command {
	command := cli.Command{
		Name:  "project",
		Usage: "options for project",
		Subcommands: []cli.Command{
			{
				Name:      "create",
				Usage:     "Create a new project",
				ArgsUsage: "<project-name>",
				Description: "Create a new project within a tenant and assigns some or all of its tenant quota.\n" +
					"   Only system administrators can create new projects. If default-router-private-ip-cidr\n" +
					"   option is omitted, it will use 192.168.0.0/16 as default router's private IP CIDR.\n" +
					"   A quota for the project can be defined during project creation and it is defined by\n" +
					"   a set of maximum resource costs. Each usage has a type, a numnber (e.g. 1) and a unit (e.g. GB).\n" +
					"   You must specify at least one cost Valid units: GB, MB, KB, B, or COUNT\n\n" +
					"   Common costs:\n" +
					"     vm.count:                     Total number of VMs (use with COUNT)\n" +
					"     vm.cpu:                       Total number of vCPUs for a VM (use with COUNT)\n" +
					"     vm.memory:                    Total amount of RAM for a VM (use with GB, MB, KB, or B)\n" +
					"     ephemeral-disk.capacity:      Total ephemeral disk capacity (use with GB, MB, KB, or B)\n" +
					"     persistent-disk.capacity:     Total persistent disk capacity (use with GB, MB, KB, or B)\n" +
					"     ephemeral-disk.count:         Number of ephemeral disks (use with COUNT)\n" +
					"     persistent-disk.count:        Number of persistent disks (use with COUNT)\n" +
					"     sdn.floatingip.size:          Number of floating ip \n\n" +
					"   Example:\n" +
					"      Set project quota with 100 VMs, 1000 GB of RAM and 500 vCPUs:\n" +
					"        photon project create project1 --tenant tenant1 \\ \n" +
					"          --limits 'vm.count 100 COUNT,\n" +
					"                    vm.cost 1000 COUNT,\n" +
					"                    vm.memory 1000 GB,\n" +
					"                    vm.cpu 500 COUNT, \n" +
					"                    ephemeral-disk 1000 COUNT,\n" +
					"                    ephemeral-disk.capacity 1000 GB,\n" +
					"                    ephemeral-disk.cost 1000 GB,\n" +
					"                    persistent-disk 1000 COUNT,\n" +
					"                    persistent-disk.capacity 1000 GB,\n" +
					"                    persistent-disk.cost 1000 GB, \n" +
					"                    storage.LOCAL_VMFS 1000 COUNT,\n" +
					"                    storage.VSAN 1000 COUNT,\n" +
					"                    sdn.floatingip.size 1000 COUNT'\n\n" +
					"      Set project quota to 30% of its tenant quota:\n" +
					"        photon project create project2 --tenant tenant1 --percent 30",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "limits, l",
						Usage: "Project limits (key value unit)",
					},
					cli.Float64Flag{
						Name:  "percent, p",
						Usage: "Project limits (0 to 100 percent of tenant quota)",
					},
					cli.StringFlag{
						Name:  "tenant, t",
						Usage: "Tenant name for project",
					},
					cli.StringFlag{
						Name:  "security-groups, g",
						Usage: "Security Groups for project",
					},
					cli.StringFlag{
						Name:  "default-router-private-ip-cidr, c",
						Usage: "Private IP range of the default router in CIDR format. Default value: 192.168.0.0/16",
					},
				},
				Action: func(c *cli.Context) {
					err := createProject(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:        "delete",
				Usage:       "Delete project with specified id",
				ArgsUsage:   "<project-id>",
				Description: "Delete a project. You must be a system administrator to delete a project.",
				Action: func(c *cli.Context) {
					err := deleteProject(c)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:      "show",
				Usage:     "Show project info with specified id",
				ArgsUsage: "<project-id>",
				Action: func(c *cli.Context) {
					err := showProject(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:      "get",
				Usage:     "Show default project.",
				ArgsUsage: " ",
				Description: "Show default project in use for photon CLI commands. Most command allow you to either\n" +
					"   use this default or specify a specific project to use.",
				Action: func(c *cli.Context) {
					err := getProject(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:      "set",
				Usage:     "Set default project",
				ArgsUsage: "<project-name>",
				Description: "Set the default project that will be used for all photon CLI commands that need a project.\n" +
					"   Most commands allow you to override the default.",
				Action: func(c *cli.Context) {
					err := setProject(c)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:      "list",
				Usage:     "List all projects",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "tenant, t",
						Usage: "Tenant name for project",
					},
				},
				Action: func(c *cli.Context) {
					err := listProjects(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:      "tasks",
				Usage:     "List all tasks related to a given project",
				ArgsUsage: "<project-id>",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "state, s",
						Usage: "specify task state for filtering",
					},
					cli.StringFlag{
						Name:  "kind, k",
						Usage: "specify task kind for filtering",
					},
				},
				Action: func(c *cli.Context) {
					err := getProjectTasks(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:      "set-security-groups",
				Usage:     "Set security groups for a project",
				ArgsUsage: "<project-id> <comma separated list of groups>",
				Description: "Set the list of Lightwave groups that can use this project. This may only be\n" +
					"   be set by a member of the project. Be cautious--you can remove your own access if you specify\n" +
					"   the wrong set of groups.\n\n" +
					"   A security group specifies both the Lightwave domain and Lightwave group.\n" +
					"   For example, a security group may be photon.vmware.com\\group-1\n\n" +
					"   Example: photon project 3f78619d-20b1-4b86-a7a6-5a9f09e59ef6 set-security-groups 'photon.vmware.com\\group-1,photon.vmware.com\\group-2'",
				Action: func(c *cli.Context) {
					err := setSecurityGroupsForProject(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Hidden:      true,
				Name:        "set_security_groups",
				Usage:       "Set security groups for a project",
				ArgsUsage:   "<project-id> <comma separated list of groups>",
				Description: "Deprecated, use set-security-groups instead",
				Action: func(c *cli.Context) {
					err := setSecurityGroupsForProject(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "iam",
				Usage: "options for identity and access management",
				Subcommands: []cli.Command{
					{
						Name:      "show",
						Usage:     "Show the IAM policy associated with a project",
						ArgsUsage: "<project-id>",
						Action: func(c *cli.Context) {
							err := getProjectIam(c)
							if err != nil {
								log.Fatal("Error: ", err)
							}
						},
					},
					{
						Name:      "add",
						Usage:     "Grant a role to a user or group on a project",
						ArgsUsage: "<project-id>",
						Description: "Grant a role to a user or group on a project. \n\n" +
							"   Example: \n" +
							"   photon project iam add <project-id> -p user1@photon.local -r contributor\n" +
							"   photon project iam add <project-id> -p photon.local\\group1 -r viewer",
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:  "principal, p",
								Usage: "User or group",
							},
							cli.StringFlag{
								Name:  "role, r",
								Usage: "'owner', 'contributor' and 'viewer'",
							},
						},
						Action: func(c *cli.Context) {
							err := modifyProjectIamPolicy(c, os.Stdout, "ADD")
							if err != nil {
								log.Fatal("Error: ", err)
							}
						},
					},
					{
						Name:      "remove",
						Usage:     "Remove a role from a user or group on a project",
						ArgsUsage: "<project-id>",
						Description: "Remove a role from a user or group on a project. \n\n" +
							"   Example: \n" +
							"   photon project iam remove <project-id> -p user1@photon.local -r contributor \n" +
							"   photon project iam remove <project-id> -p photon.local\\group1 -r viewer",
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:  "principal, p",
								Usage: "User or group",
							},
							cli.StringFlag{
								Name:  "role, r",
								Usage: "'owner', 'contributor' and 'viewer'. Or use '*' to remove all existing roles.",
							},
						},
						Action: func(c *cli.Context) {
							err := modifyProjectIamPolicy(c, os.Stdout, "REMOVE")
							if err != nil {
								log.Fatal("Error: ", err)
							}
						},
					},
				},
			},
			// Load Project Quota related logic from separated file.
			getProjectQuotaCommand(),
		},
	}
	return command
}

// Sends a create project task to client based on the cli.Context
// Returns an error if one occurred
func createProject(c *cli.Context, w io.Writer) error {
	err := checkArgCount(c, 1)
	if err != nil {
		return err
	}
	name := c.Args().First()
	tenantName := c.String("tenant")
	limits := c.String("limits")
	percent := c.Float64("percent") / 100.0
	securityGroups := c.String("security-groups")
	defaultRouterPrivateIpCidr := c.String("default-router-private-ip-cidr")

	if len(defaultRouterPrivateIpCidr) == 0 {
		defaultRouterPrivateIpCidr = "192.168.0.0/16"
	}

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	tenant, err := verifyTenant(tenantName)
	if err != nil {
		return err
	}

	// Get project quota if present
	var limitsList []photon.QuotaLineItem
	if c.IsSet("limits") && c.IsSet("percent") {
		return fmt.Errorf("Error: Can only specify one of '--limits' or '--percent'")
	}

	if c.IsSet("limits") {
		percent = 0.0
		limitsList, err = parseLimitsListFromFlag(limits)
		if err != nil {
			return err
		}
	}

	if c.IsSet("percent") {
		limitsList = []photon.QuotaLineItem{
			{Key: "subdivide.percent", Value: percent, Unit: "COUNT"}}
	}

	if !c.GlobalIsSet("non-interactive") {
		name, err = askForInput("Project name: ", name)
		if err != nil {
			return err
		}

		limitsList, err = askForLimitList(limitsList)
		if err != nil {
			return err
		}
	}

	projectSpec := photon.ProjectCreateSpec{}
	projectSpec.Name = name
	projectSpec.DefaultRouterPrivateIpCidr = defaultRouterPrivateIpCidr

	if !c.GlobalIsSet("non-interactive") {
		fmt.Printf("\nTenant name: %s\n", tenant.Name)
		fmt.Printf("Creating project name: %s\n\n", name)
		fmt.Println("Please make sure limits below are correct:")
		for i, l := range limitsList {
			fmt.Printf("%d: %s, %g, %s\n", i+1, l.Key, l.Value, l.Unit)
		}
	}
	if confirmed(c) {
		if len(securityGroups) > 0 {
			projectSpec.SecurityGroups = regexp.MustCompile(`\s*,\s*`).Split(securityGroups, -1)
		}

		projectQuota := photon.Quota{}
		if percent > 0 {
			tenatQuota, err := client.Photonclient.Tenants.GetQuota(tenant.ID)
			if err != nil {
				return err
			}

			quotaSpec := photon.QuotaSpec{}

			if tenatQuota != nil {
				for key, element := range tenatQuota.QuotaLineItems {
					quotaSpec[key] = photon.QuotaStatusLineItem{Unit: element.Unit, Limit: element.Limit * percent, Usage: 0}
				}

				projectQuota.QuotaLineItems = quotaSpec
			}
		} else if limitsList != nil {
			quotaSpec := convertQuotaSpecFromQuotaLineItems(limitsList)
			projectQuota.QuotaLineItems = quotaSpec
		}

		projectSpec.ResourceQuota = projectQuota

		createTask, err := client.Photonclient.Tenants.CreateProject(tenant.ID, &projectSpec)
		if err != nil {
			return err
		}

		id, err := waitOnTaskOperation(createTask.ID, c)
		if err != nil {
			return err
		}

		if utils.NeedsFormatting(c) {
			project, err := client.Photonclient.Projects.Get(id)
			if err != nil {
				return err
			}
			utils.FormatObject(project, w, c)
		}
	} else {
		fmt.Println("OK. Canceled")
	}

	return nil
}

// Sends a delete project task to client based on the cli.Context
// Returns an error if one occurred
func deleteProject(c *cli.Context) error {
	err := checkArgCount(c, 1)
	if err != nil {
		return err
	}
	id := c.Args().First()

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	deleteTask, err := client.Photonclient.Projects.Delete(id)
	if err != nil {
		return err
	}
	_, err = waitOnTaskOperation(deleteTask.ID, c)
	if err != nil {
		return err
	}

	err = clearConfigProject(id)
	if err != nil {
		return err
	}

	return nil
}

// Show project info with the specified project id, returns an error if one occurred
func showProject(c *cli.Context, w io.Writer) error {
	err := checkArgCount(c, 1)
	if err != nil {
		return err
	}
	id := c.Args().First()

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	project, err := client.Photonclient.Projects.Get(id)
	if err != nil {
		return err
	}

	if c.GlobalIsSet("non-interactive") {
		securityGroups := []string{}
		for _, s := range project.SecurityGroups {
			securityGroups = append(securityGroups, fmt.Sprintf("%s:%t", s.Name, s.Inherited))
		}
		scriptSecurityGroups := strings.Join(securityGroups, ",")
		quotaString := quotaSpecToString(project.ResourceQuota.QuotaLineItems)

		fmt.Printf("%s\t%s\t%s\t%s\n", project.ID, project.Name,
			quotaString, scriptSecurityGroups)
	} else if utils.NeedsFormatting(c) {
		utils.FormatObject(project, w, c)
	} else {
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 4, 4, 2, ' ', 0)
		fmt.Fprintf(w, "Project ID: %s\n", project.ID)
		fmt.Fprintf(w, "  Name: %s\n", project.Name)
		fmt.Fprintf(w, "    Limits:\n")
		for k, l := range project.ResourceQuota.QuotaLineItems {
			fmt.Fprintf(w, "      %s\t%g\t%s\n", k, l.Limit, l.Unit)
		}
		fmt.Fprintf(w, "    Usage:\n")
		for k, u := range project.ResourceQuota.QuotaLineItems {
			fmt.Fprintf(w, "      %s\t%g\t%s\n", k, u.Usage, u.Unit)
		}
		if len(project.SecurityGroups) != 0 {
			fmt.Fprintf(w, "  SecurityGroups:\n")
			for _, s := range project.SecurityGroups {
				fmt.Fprintf(w, "    %s\t%t\n", s.Name, s.Inherited)
			}
		}
		err = w.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}

// Sends a get project task to client based on the config file
// Returns an error if one occurred
func getProject(c *cli.Context, w io.Writer) error {
	err := checkArgCount(c, 0)
	if err != nil {
		return err
	}
	config, err := cf.LoadConfig()
	if err != nil {
		return err
	}

	project := config.Project
	if project == nil {
		return fmt.Errorf("Error: No Project selected\n")
	}

	if c.GlobalIsSet("non-interactive") {
		fmt.Printf("%s\t%s\n", project.Name, project.ID)
	} else if utils.NeedsFormatting(c) {
		utils.FormatObject(project, w, c)
	} else {
		fmt.Printf("Current project name is '%s', with ID '%s' \n", project.Name, project.ID)
	}
	return nil
}

// Set project name and id to config file
// Returns an error if one occurred
func setProject(c *cli.Context) error {
	err := checkArgCount(c, 1)
	if err != nil {
		return err
	}
	name := c.Args().First()

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	config, err := cf.LoadConfig()
	if err != nil {
		return err
	}

	if config == nil || config.Tenant == nil {
		return fmt.Errorf("Error: Set tenant first using 'tenant set <name>' or '-t <name>' option")
	}

	project, err := findProject(config.Tenant.ID, name)
	if err != nil {
		return err
	}

	config.Project = &cf.ProjectConfiguration{Name: project.Name, ID: project.ID}
	err = cf.SaveConfig(config)
	if err != nil {
		return err
	}

	if !c.GlobalIsSet("non-interactive") {
		fmt.Printf("Project set to '%s'\n", name)
	}

	return nil
}

// Retrieves a list of projects, returns an error if one occurred
func listProjects(c *cli.Context, w io.Writer) error {
	err := checkArgCount(c, 0)
	if err != nil {
		return err
	}
	tenantName := c.String("tenant")

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	tenant, err := verifyTenant(tenantName)
	if err != nil {
		return err
	}

	projects, err := client.Photonclient.Tenants.GetProjects(tenant.ID, nil)
	if err != nil {
		return err
	}

	if c.GlobalIsSet("non-interactive") {
		for _, t := range projects.Items {
			quotaString := quotaSpecToString(t.ResourceQuota.QuotaLineItems)
			fmt.Printf("%s\t%s\t%s\n", t.ID, t.Name, quotaString)
		}
	} else if utils.NeedsFormatting(c) {
		utils.FormatObjects(projects.Items, w, c)
	} else {
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 4, 4, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tName\tLimit\tUsage\n")
		for _, t := range projects.Items {
			qt := t.ResourceQuota
			if len(qt.QuotaLineItems) == 0 {
				fmt.Fprintf(w, "%s\t%s\n", t.ID, t.Name)
			} else {
				count := 0
				for k, v := range qt.QuotaLineItems {
					if count == 0 {
						fmt.Fprintf(w, "%s\t%s\t%s %g %s\t%s %g %s\n", t.ID, t.Name,
							k, v.Limit, v.Unit, k, v.Usage, v.Unit)
					} else {
						fmt.Fprintf(w, "\t\t%s %g %s\t%s %g %s\n",
							k, v.Limit, v.Unit, k, v.Usage, v.Unit)
					}
					count++
				}
			}
		}
		err := w.Flush()
		if err != nil {
			return err
		}
		fmt.Printf("\nTotal projects: %d\n", len(projects.Items))
	}
	return nil
}

// Retrieves tasks for project
func getProjectTasks(c *cli.Context, w io.Writer) error {
	err := checkArgCount(c, 1)
	if err != nil {
		return err
	}
	id := c.Args().First()
	state := c.String("state")
	kind := c.String("kind")

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	options := &photon.TaskGetOptions{
		State: state,
		Kind:  kind,
	}

	taskList, err := client.Photonclient.Projects.GetTasks(id, options)
	if err != nil {
		return err
	}

	err = printTaskList(taskList.Items, c)
	if err != nil {
		return err
	}

	return nil
}

// Set security groups for a project
func setSecurityGroupsForProject(c *cli.Context) error {
	err := checkArgCount(c, 2)
	if err != nil {
		return err
	}
	id := c.Args().First()
	securityGroups := &photon.SecurityGroupsSpec{
		Items: regexp.MustCompile(`\s*,\s*`).Split(c.Args()[1], -1),
	}
	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	task, err := client.Photonclient.Projects.SetSecurityGroups(id, securityGroups)
	if err != nil {
		return err
	}

	_, err = waitOnTaskOperation(task.ID, c)
	if err != nil {
		return err
	}

	return nil
}

// Retrieves IAM Policy for specified project
func getProjectIam(c *cli.Context) error {
	err := checkArgCount(c, 1)
	if err != nil {
		return err
	}
	id := c.Args().First()

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	policy, err := client.Photonclient.Projects.GetIam(id)
	if err != nil {
		return err
	}

	err = printIamPolicy(*policy, c)
	if err != nil {
		return err
	}

	return nil
}

// Grant or remove a role from a principal on the specified project
func modifyProjectIamPolicy(c *cli.Context, w io.Writer, action string) error {
	err := checkArgCount(c, 1)
	if err != nil {
		return err
	}
	projectID := c.Args()[0]
	principal := c.String("principal")
	role := c.String("role")

	if !c.GlobalIsSet("non-interactive") {
		var err error
		principal, err = askForInput("Principal: ", principal)
		if err != nil {
			return err
		}
	}

	if len(principal) == 0 {
		return fmt.Errorf("Please provide principal")
	}

	if !c.GlobalIsSet("non-interactive") {
		var err error
		role, err = askForInput("Role: ", role)
		if err != nil {
			return err
		}
	}

	if len(role) == 0 {
		return fmt.Errorf("Please provide role")
	}

	client.Photonclient, err = client.GetClient(c)
	if err != nil {
		return err
	}

	var delta photon.PolicyDelta
	delta = photon.PolicyDelta{Principal: principal, Action: action, Role: role}
	task, err := client.Photonclient.Projects.ModifyIam(projectID, &delta)

	if err != nil {
		return err
	}

	_, err = waitOnTaskOperation(task.ID, c)
	if err != nil {
		return err
	}

	if utils.NeedsFormatting(c) {
		policy, err := client.Photonclient.Projects.GetIam(projectID)
		if err != nil {
			return err
		}
		utils.FormatObject(policy, w, c)
	}

	return nil
}
