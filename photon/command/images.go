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
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/vmware/photon-controller-cli/photon/client"
	"github.com/vmware/photon-controller-cli/photon/utils"

	"github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/vmware/photon-controller-go-sdk/photon"
)

// Creates a cli.Command for images
// Subcommands: create; Usage: image create <path> [<options>]
//              delete; Usage: image delete <id>
//              list;   Usage: image list
//              show;   Usage: image show <id>
//              tasks;  Usage: image tasks <id> [<options>]
func GetImagesCommand() cli.Command {
	command := cli.Command{
		Name:  "image",
		Usage: "options for image",
		Subcommands: []cli.Command{
			{
				Name:  "create",
				Usage: "Create a new image",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Image name",
					},
					cli.StringFlag{
						Name:  "image_replication, i",
						Usage: "Image replication type",
					},
				},
				Action: func(c *cli.Context) {
					err := createImage(c, os.Stdout)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "delete",
				Usage: "delete an image",
				Action: func(c *cli.Context) {
					err := deleteImage(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "list",
				Usage: "list images",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Image name",
					},
				},
				Action: func(c *cli.Context) {
					err := listImages(c, os.Stdout)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "show",
				Usage: "show an image",
				Action: func(c *cli.Context) {
					err := showImage(c, os.Stdout)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
			{
				Name:  "tasks",
				Usage: "Show image tasks",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "state, s",
						Usage: "Filter by task state",
					},
				},
				Action: func(c *cli.Context) {
					err := getImageTasks(c)
					if err != nil {
						log.Fatal("Error: ", err)
					}
				},
			},
		},
	}
	return command
}

// Create an image
func createImage(c *cli.Context, w io.Writer) error {
	if len(c.Args()) > 1 {
		return fmt.Errorf("Unknown argument: %v", c.Args()[1:])
	}
	path := c.Args().First()
	name := c.String("name")
	replicationType := c.String("image_replication")

	if !utils.IsNonInteractive(c) {
		var err error
		path, err = askForInput("Image path: ", path)
		if err != nil {
			return err
		}
	}

	if len(path) == 0 {
		return fmt.Errorf("Please provide image path")
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	_, err = os.Stat(path)
	if err != nil {
		return fmt.Errorf("No such image file at that path")
	}

	if !c.GlobalIsSet("non-interactive") {
		defaultName := path
		name, err = askForInput("Image name (default: "+defaultName+"): ", name)
		if err != nil {
			return err
		}

		if len(name) == 0 {
			name = defaultName
		}

		defaultReplication := "EAGER"
		replicationType, err = askForInput("Image replication type (default: "+defaultReplication+"): ", replicationType)
		if err != nil {
			return err
		}
		if len(replicationType) == 0 {
			replicationType = defaultReplication
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	options := &photon.ImageCreateOptions{
		ReplicationType: replicationType,
	}
	if len(replicationType) == 0 {
		options = nil
	}

	uploadTask, err := client.Esxclient.Images.Create(file, name, options)
	if err != nil {
		return err
	}

	imageID, err := waitOnTaskOperation(uploadTask.ID, c)
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}

	if utils.NeedsFormatting(c) {
		image, err := client.Esxclient.Images.Get(imageID)
		if err != nil {
			return err
		}
		utils.FormatObject(image, w, c)
	}

	return nil
}

// Deletes an image by id
func deleteImage(c *cli.Context) error {
	err := checkArgNum(c.Args(), 1, "image delete <path>")
	if err != nil {
		return err
	}
	id := c.Args().First()

	if confirmed(utils.IsNonInteractive(c)) {
		client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
		if err != nil {
			return err
		}

		deleteTask, err := client.Esxclient.Images.Delete(id)
		if err != nil {
			return err
		}

		_, err = waitOnTaskOperation(deleteTask.ID, c)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("OK, canceled")
	}

	return nil
}

// Lists all images
func listImages(c *cli.Context, w io.Writer) error {
	err := checkArgNum(c.Args(), 0, "image list [<options>]")
	if err != nil {
		return err
	}
	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	name := c.String("name")
	options := &photon.ImageGetOptions{
		Name: name,
	}
	images, err := client.Esxclient.Images.GetAll(options)
	if err != nil {
		return err
	}

	if c.GlobalIsSet("non-interactive") {
		for _, image := range images.Items {
			fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\t%s\n", image.ID, image.Name, image.State, image.Size,
				image.ReplicationType, image.ReplicationProgress, image.SeedingProgress)
		}
	} else if utils.NeedsFormatting(c) {
		utils.FormatObjects(images.Items, w, c)
	} else {
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 4, 4, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tName\tState\tSize(Byte)\tReplication_type\tReplicationProgress\tSeedingProgress\n")
		for _, image := range images.Items {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n", image.ID, image.Name, image.State, image.Size,
				image.ReplicationType, image.ReplicationProgress, image.SeedingProgress)
		}
		err = w.Flush()
		if err != nil {
			return err
		}
		fmt.Printf("\nTotal: %d\n", len(images.Items))
	}

	return nil
}

// Shows an image based on id
func showImage(c *cli.Context, w io.Writer) error {
	id := c.Args().First()

	if !utils.IsNonInteractive(c) {
		var err error
		id, err = askForInput("Image id: ", id)
		if err != nil {
			return err
		}
	}

	if len(id) == 0 {
		return fmt.Errorf("Please provide image id")
	}

	var err error
	client.Esxclient, err = client.GetClient(utils.IsNonInteractive(c))
	if err != nil {
		return err
	}

	image, err := client.Esxclient.Images.Get(id)
	if err != nil {
		return err
	}

	if c.GlobalIsSet("non-interactive") {
		settings := []string{}
		for _, setting := range image.Settings {
			settings = append(settings, fmt.Sprintf("%s:%s", setting.Name, setting.DefaultValue))
		}
		scriptSettings := strings.Join(settings, ",")
		fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\n", image.ID, image.Name, image.State, image.Size, image.ReplicationType,
			image.ReplicationProgress, image.SeedingProgress, scriptSettings)

	} else if utils.NeedsFormatting(c) {
		utils.FormatObject(image, w, c)
	} else {
		fmt.Printf("Image ID: %s\n", image.ID)
		fmt.Printf("  Name:                       %s\n", image.Name)
		fmt.Printf("  State:                      %s\n", image.State)
		fmt.Printf("  Size:                       %d Byte(s)\n", image.Size)
		fmt.Printf("  Image Replication Type:     %s\n", image.ReplicationType)
		fmt.Printf("  Image Replication Progress: %s\n", image.ReplicationProgress)
		fmt.Printf("  Image Seeding Progress:     %s\n", image.SeedingProgress)
		fmt.Printf("  Settings: \n")
		for _, setting := range image.Settings {
			fmt.Printf("    %s : %s\n", setting.Name, setting.DefaultValue)
		}
	}

	return nil
}

// Retrieves tasks from specified image
func getImageTasks(c *cli.Context) error {
	err := checkArgNum(c.Args(), 1, "image tasks <id> [<options>]")
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

	taskList, err := client.Esxclient.Images.GetTasks(id, options)
	if err != nil {
		return err
	}

	err = printTaskList(taskList.Items, c)
	if err != nil {
		return err
	}

	return nil
}
