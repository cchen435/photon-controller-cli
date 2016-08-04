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

	"github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/vmware/photon-controller-go-sdk/photon"
)

// Creates a cli.Command for tenant
// Subcommands: create; Usage: tenant create <name>
//              delete; Usage: tenant delete <id>
//              show;   Usage: tenant show <id>
//              list;   Usage: tenant list
//              set;    Usage: tenant set <name>
//              get;    Usage: tenant get
//              tasks;  Usage: tenant tasks <id> [<options>]
func GetTenantsCommand() cli.Command {
	command := cli.Command{
		Name:  "tenant",
		Usage: "options for tenant",
		Subcommands: []cli.Command{
			{
				Name:  "create",
				Usage: "Create a new tenant",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "security-groups, s",
						Usage: "Comma-separated security group names",
					},
				},
				Action: func(c *cli.Context) {
					err := createTenant(c, os.Stdout)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "delete",
				Usage: "Delete a tenant",
				Action: func(c *cli.Context) {
					err := deleteTenant(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "show",
				Usage: "Show tenant info with specified id",
				Action: func(c *cli.Context) {
					err := showTenant(c, os.Stdout)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:  "list",
				Usage: "List tenants",
				Action: func(c *cli.Context) {
					err := listTenants(c, os.Stdout)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "set",
				Usage: "Select tenant to work with",
				Action: func(c *cli.Context) {
					err := setTenant(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "get",
				Usage: "Get current tenant",
				Action: func(c *cli.Context) {
					err := getTenant(c, os.Stdout)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "tasks",
				Usage: "Show tenant tasks",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "state, s",
						Usage: "Filter by task sate",
					},
				},
				Action: func(c *cli.Context) {
					err := getTenantTasks(c, os.Stdout)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "set_security_groups",
				Usage: "Set security groups for a tenant",
				Action: func(c *cli.Context) {
					err := setSecurityGroups(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
		},
	}
	return command
}

// Sends a create tenant task to client based on the cli.Context
// Returns an error if one occurred
func createTenant(c *cli.Context, w io.Writer) error {
	if len(c.Args()) > 1 {
		return fmt.Errorf("Unknown argument: %v", c.Args()[1:])
	}
	name := c.Args().First()
	securityGroups := c.String("security-groups")

	if !utils.IsNonInteractive(c) {
		var err error
		name, err = askForInput("Tenant name: ", name)
		if err != nil {
			return err
		}
		securityGroups, err =
			askForInput("Comma-separated security group names, or hit enter for no security groups): ",
				securityGroups)
		if err != nil {
			return err
		}
	}

	if len(name) == 0 {
		return fmt.Errorf("Please provide tenant name")
	}
	securityGroupList := []string{}
	if securityGroups != "" {
		securityGroupList = regexp.MustCompile(`\s*,\s*`).Split(securityGroups, -1)
	}

	tenantSpec := &photon.TenantCreateSpec{
		Name:           name,
		SecurityGroups: securityGroupList,
	}

	var err error
	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	createTask, err := client.Esxclient.Tenants.Create(tenantSpec)

	if err != nil {
		return err
	}

	id, err := waitOnTaskOperation(createTask.ID, c)
	if err != nil {
		return err
	}

	if utils.NeedsFormatting(c) {
		tenant, err := client.Esxclient.Tenants.Get(id)
		if err != nil {
			return err
		}
		utils.FormatObject(tenant, w, c)
	}

	return nil
}

// Retrieves a list of tenants, returns an error if one occurred
func listTenants(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 0, "tenant list")
	if err != nil {
		return err
	}
	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	tenants, err := client.Esxclient.Tenants.GetAll()
	if err != nil {
		return err
	}

	if c.GlobalIsSet("non-interactive") {
		for _, tenant := range tenants.Items {
			fmt.Printf("%s\t%s\n", tenant.ID, tenant.Name)
		}
	} else if utils.NeedsFormatting(c) {
		utils.FormatObjects(tenants.Items, w, c)
	} else {
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 4, 4, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tName\n")
		for _, tenant := range tenants.Items {
			fmt.Fprintf(w, "%s\t%s\n", tenant.ID, tenant.Name)
		}
		err = w.Flush()
		if err != nil {
			return err
		}
		fmt.Printf("\nTotal: %d\n", len(tenants.Items))
	}

	return nil
}

// Sends a delete tenant task to client based on the cli.Context
// Returns an error if one occurred
func deleteTenant(c *cli.Context) error {
	err := checkArgNum(c.Args(), 1, "tenant delete <id>")
	if err != nil {
		return err
	}
	id := c.Args().First()

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	deleteTask, err := client.Esxclient.Tenants.Delete(id)
	if err != nil {
		return err
	}

	_, err = waitOnTaskOperation(deleteTask.ID, c)
	if err != nil {
		return err
	}

	err = clearConfigTenant(id)
	if err != nil {
		return err
	}

	return nil
}

func showTenant(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 1, "tenant show <id>")
	if err != nil {
		return err
	}
	id := c.Args().First()

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	tenant, err := client.Esxclient.Tenants.Get(id)
	if err != nil {
		return err
	}

	if c.GlobalIsSet("non-interactive") {
		securityGroups := []string{}
		for _, s := range tenant.SecurityGroups {
			securityGroups = append(securityGroups, fmt.Sprintf("%s:%t", s.Name, s.Inherited))
		}
		scriptSecurityGroups := strings.Join(securityGroups, ",")
		fmt.Printf("%s\t%s\t%s\n", tenant.ID, tenant.Name, scriptSecurityGroups)
	} else if utils.NeedsFormatting(c) {
		utils.FormatObject(tenant, w, c)
	} else {
		fmt.Println("Tenant ID: ", tenant.ID)
		fmt.Println("  Name:              ", tenant.Name)
		for i, s := range tenant.SecurityGroups {
			fmt.Printf("    SecurityGroups %d:\n", i+1)
			fmt.Println("      Name:          ", s.Name)
			fmt.Println("      Inherited:     ", s.Inherited)
		}
	}

	return nil
}

// Overwrites the tenant in the config file
func setTenant(c *cli.Context) error {
	err := checkArgNum(c.Args(), 1, "tenant set <name>")
	if err != nil {
		return err
	}
	name := c.Args().First()

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	// Ensure tenant exists
	id, err := findTenantID(name)
	if len(id) == 0 || err != nil {
		return err
	}

	config, err := cf.LoadConfig()
	if err != nil {
		return err
	}

	config.Tenant = &cf.TenantConfiguration{Name: name, ID: id}
	err = cf.SaveConfig(config)
	if err != nil {
		return err
	}

	err = clearConfigProject("")
	if err != nil {
		return err
	}

	if !utils.IsNonInteractive(c) {
		fmt.Printf("Tenant set to '%s'\n", name)
	}
	return nil
}

// Outputs the set tenant otherwise informs user it is not set
func getTenant(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 0, "tenant get")
	if err != nil {
		return err
	}
	config, err := cf.LoadConfig()
	if err != nil {
		return err
	}

	tenant := config.Tenant
	if tenant == nil {
		fmt.Printf("No tenant selected\n")
	} else {
		if c.GlobalIsSet("non-interactive") {
			fmt.Printf("%s\t%s\n", tenant.ID, tenant.Name)
		} else if utils.NeedsFormatting(c) {
			utils.FormatObject(tenant, w, c)
		} else {
			fmt.Printf("Current tenant is '%s' with ID %s\n", tenant.Name, tenant.ID)
		}
	}
	return nil
}

// Retrieves tasks from specified tenant
func getTenantTasks(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 1, "tenant tasks <id> [<options>]")
	if err != nil {
		return err
	}
	id := c.Args().First()

	state := c.String("state")
	options := &photon.TaskGetOptions{
		State: state,
	}

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	taskList, err := client.Esxclient.Tenants.GetTasks(id, options)
	if err != nil {
		return err
	}

	err = printTaskList(taskList.Items, c)
	if err != nil {
		return err
	}
	return nil
}

// Set security groups for a tenant
func setSecurityGroups(c *cli.Context) error {
	err := checkArgNum(c.Args(), 2, "tenant set_security_groups <id> <comma-separated security group names>")
	if err != nil {
		return err
	}
	id := c.Args().First()
	items := []string{}
	if c.Args()[1] != "" {
		items = regexp.MustCompile(`\s*,\s*`).Split(c.Args()[1], -1)
	}
	securityGroups := &photon.SecurityGroupsSpec{
		Items: items,
	}

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	task, err := client.Esxclient.Tenants.SetSecurityGroups(id, securityGroups)
	if err != nil {
		return err
	}

	_, err = waitOnTaskOperation(task.ID, c)
	if err != nil {
		return err
	}

	return nil
}
