package ddcloud

import (
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"time"
)

const (
	resourceKeyServerName            = "name"
	resourceKeyServerDescription     = "description"
	resourceKeyServerAdminPassword   = "admin_password"
	resourceKeyServerNetworkDomainID = "networkdomain"
	resourceKeyServerMemoryGB        = "memory_gb"
	resourceKeyServerCPUCount        = "cpu_count"
	resourceKeyServerImageDisk       = "image_disk"
	resourceKeyServerAdditionalDisk  = "additional_disk"
	resourceKeyServerDiskID          = "disk_id"
	resourceKeyServerDiskSizeGB      = "size_gb"
	resourceKeyServerDiskUnitID      = "scsi_unit_id"
	resourceKeyServerDiskSpeed       = "speed"
	resourceKeyServerOSImageID       = "osimage_id"
	resourceKeyServerOSImageName     = "osimage_name"
	resourceKeyServerPrimaryVLAN     = "primary_adapter_vlan"
	resourceKeyServerPrimaryIPv4     = "primary_adapter_ipv4"
	resourceKeyServerPrimaryIPv6     = "primary_adapter_ipv6"
	resourceKeyServerPrimaryDNS      = "dns_primary"
	resourceKeyServerSecondaryDNS    = "dns_secondary"
	resourceKeyServerAutoStart       = "auto_start"
	resourceKeyServerTag             = "tag"
	resourceKeyServerTagName         = "name"
	resourceKeyServerTagValue        = "value"
	resourceCreateTimeoutServer      = 30 * time.Minute
	resourceUpdateTimeoutServer      = 10 * time.Minute
	resourceDeleteTimeoutServer      = 15 * time.Minute
	serverShutdownTimeout            = 5 * time.Minute
)

func resourceServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceServerCreate,
		Read:   resourceServerRead,
		Update: resourceServerUpdate,
		Delete: resourceServerDelete,

		Schema: map[string]*schema.Schema{
			resourceKeyServerName: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			resourceKeyServerDescription: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			resourceKeyServerAdminPassword: &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			resourceKeyServerMemoryGB: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerCPUCount: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			// TODO: Merge "image_disk" and "additional_disk" so there's just "disk".
			resourceKeyServerImageDisk:      schemaServerDisk(),
			resourceKeyServerAdditionalDisk: schemaServerDisk(),
			resourceKeyServerNetworkDomainID: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			resourceKeyServerPrimaryVLAN: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerPrimaryIPv4: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerPrimaryIPv6: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			resourceKeyServerPrimaryDNS: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			resourceKeyServerSecondaryDNS: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			resourceKeyServerOSImageID: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerOSImageName: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerAutoStart: &schema.Schema{
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
				Default:  false,
			},
			resourceKeyServerTag: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Default:  nil,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resourceKeyServerTagName: &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						resourceKeyServerTagValue: &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: hashServerTag,
			},
		},
	}
}

// Create a server resource.
func resourceServerCreate(data *schema.ResourceData, provider interface{}) error {
	name := data.Get(resourceKeyServerName).(string)
	description := data.Get(resourceKeyServerDescription).(string)
	adminPassword := data.Get(resourceKeyServerAdminPassword).(string)
	networkDomainID := data.Get(resourceKeyServerNetworkDomainID).(string)
	primaryDNS := data.Get(resourceKeyServerPrimaryDNS).(string)
	secondaryDNS := data.Get(resourceKeyServerSecondaryDNS).(string)
	autoStart := data.Get(resourceKeyServerAutoStart).(bool)

	log.Printf("Create server '%s' in network domain '%s' (description = '%s').", name, networkDomainID, description)

	apiClient := provider.(*providerState).Client()

	networkDomain, err := apiClient.GetNetworkDomain(networkDomainID)
	if err != nil {
		return err
	}

	if networkDomain == nil {
		return fmt.Errorf("No network domain was found with Id '%s'.", networkDomainID)
	}

	dataCenterID := networkDomain.DatacenterID
	log.Printf("Server will be deployed in data centre '%s'.", dataCenterID)

	propertyHelper := propertyHelper(data)

	// Retrieve image details.
	osImageID := propertyHelper.GetOptionalString(resourceKeyServerOSImageID, false)
	osImageName := propertyHelper.GetOptionalString(resourceKeyServerOSImageName, false)

	var osImage *compute.OSImage
	if osImageID != nil {
		// TODO: Look up OS image by Id (first, implement in compute API client).

		return fmt.Errorf("Specifying osimage_id is not supported yet.")
	} else if osImageName != nil {
		log.Printf("Looking up OS image '%s' by name...", *osImageName)

		osImage, err = apiClient.FindOSImage(*osImageName, dataCenterID)
		if err != nil {
			return err
		}

		if osImage == nil {
			log.Printf("Warning - unable to find an OS image named '%s' in data centre '%s' (which is where the target network domain, '%s', is located).", *osImageName, dataCenterID, networkDomainID)

			return fmt.Errorf("Unable to find an OS image named '%s' in data centre '%s' (which is where the target network domain, '%s', is located).", *osImageName, dataCenterID, networkDomainID)
		}

		log.Printf("Server will be deployed from OS image with Id '%s'.", osImage.ID)
		data.Set(resourceKeyServerOSImageID, osImage.ID)
	} else {
		return fmt.Errorf("Must specify either osimage_id or osimage_name.")
	}

	deploymentConfiguration := compute.ServerDeploymentConfiguration{
		Name:                  name,
		Description:           description,
		AdministratorPassword: adminPassword,
		Start: autoStart,
	}
	err = deploymentConfiguration.ApplyImage(osImage)
	if err != nil {
		return err
	}

	// Memory and CPU
	memoryGB := propertyHelper.GetOptionalInt(resourceKeyServerMemoryGB, false)
	if memoryGB != nil {
		deploymentConfiguration.MemoryGB = *memoryGB
	} else {
		data.Set(resourceKeyServerMemoryGB, deploymentConfiguration.MemoryGB)
	}

	cpuCount := propertyHelper.GetOptionalInt(resourceKeyServerCPUCount, false)
	if cpuCount != nil {
		deploymentConfiguration.CPU.Count = *cpuCount
	} else {
		data.Set(resourceKeyServerCPUCount, deploymentConfiguration.CPU.Count)
	}

	// Network
	primaryVLANID := propertyHelper.GetOptionalString(resourceKeyServerPrimaryVLAN, false)
	primaryIPv4Address := propertyHelper.GetOptionalString(resourceKeyServerPrimaryIPv4, false)

	deploymentConfiguration.Network = compute.VirtualMachineNetwork{
		NetworkDomainID: networkDomainID,
		PrimaryAdapter: compute.VirtualMachineNetworkAdapter{
			VLANID:             primaryVLANID,
			PrivateIPv4Address: primaryIPv4Address,
		},
	}
	deploymentConfiguration.PrimaryDNS = primaryDNS
	deploymentConfiguration.SecondaryDNS = secondaryDNS

	log.Printf("Server deployment configuration: %+v", deploymentConfiguration)
	log.Printf("Server CPU deployment configuration: %+v", deploymentConfiguration.CPU)

	serverID, err := apiClient.DeployServer(deploymentConfiguration)
	if err != nil {
		return err
	}

	data.SetId(serverID)

	log.Printf("Server '%s' is being provisioned...", name)

	resource, err := apiClient.WaitForDeploy(compute.ResourceTypeServer, serverID, resourceCreateTimeoutServer)
	if err != nil {
		return err
	}

	// Capture additional properties that may only be available after deployment.
	server := resource.(*compute.Server)

	data.Partial(true)

	serverIPv4Address := server.Network.PrimaryAdapter.PrivateIPv4Address
	data.Set(resourceKeyServerPrimaryIPv4, serverIPv4Address)
	data.SetPartial(resourceKeyServerPrimaryIPv4)

	serverIPv6Address := *server.Network.PrimaryAdapter.PrivateIPv6Address
	data.Set(resourceKeyServerPrimaryIPv6, serverIPv6Address)
	data.SetPartial(resourceKeyServerPrimaryIPv6)

	// Adjust image size for image disk(s) if they were explicitly specified in the configuration.
	log.Printf("Configuring image disks for server '%s'...", serverID)
	imageDisksByUnitID := getDisksByUnitID(server.Disks)
	err = createImageDisks(imageDisksByUnitID, data, apiClient)
	if err != nil {
		return err
	}

	// Additional disks
	log.Printf("Configuring additional disks for server '%s'...", serverID)
	additionalDisks := propertyHelper.GetServerDisks(resourceKeyServerAdditionalDisk)
	if len(additionalDisks) == 0 {
		return nil
	}

	addedDisks := additionalDisks[:0]
	for index := range additionalDisks {
		disk := &additionalDisks[index]

		log.Printf("Adding disk with SCSI unit ID %d to server '%s'...", disk.SCSIUnitID, serverID)

		var diskID string
		diskID, err = apiClient.AddDiskToServer(serverID, disk.SCSIUnitID, disk.SizeGB, disk.Speed)
		if err != nil {
			return err
		}

		log.Printf("Disk with SCSI unit ID %d in server '%s' will have Id '%s'.", disk.SCSIUnitID, serverID, diskID)

		disk.ID = &diskID

		addedDisks = append(addedDisks, *disk)
		propertyHelper.SetServerDisks(resourceKeyServerAdditionalDisk, addedDisks)
		data.SetPartial(resourceKeyServerAdditionalDisk)

		_, err = apiClient.WaitForChange(
			compute.ResourceTypeServer,
			serverID,
			"Add disk",
			resourceUpdateTimeoutServer,
		)
		if err != nil {
			return err
		}

		log.Printf("Added disk with SCSI unit ID %d to server '%s' as disk '%s'.", disk.SCSIUnitID, serverID, diskID)
	}

	err = applyServerTags(data, apiClient)
	if err != nil {
		return err
	}
	data.SetPartial(resourceKeyServerTag)

	data.Partial(false)

	return nil
}

// Read a server resource.
func resourceServerRead(data *schema.ResourceData, provider interface{}) error {
	id := data.Id()
	name := data.Get(resourceKeyServerName).(string)
	description := data.Get(resourceKeyServerDescription).(string)
	networkDomainID := data.Get(resourceKeyServerNetworkDomainID).(string)

	log.Printf("Read server '%s' (Id = '%s') in network domain '%s' (description = '%s').", name, id, networkDomainID, description)

	apiClient := provider.(*providerState).Client()
	server, err := apiClient.GetServer(id)
	if err != nil {
		return err
	}

	if server == nil {
		log.Printf("Server '%s' has been deleted.", id)

		// Mark as deleted.
		data.SetId("")

		return nil
	}

	data.Set(resourceKeyServerName, server.Name)
	data.Set(resourceKeyServerDescription, server.Description)
	data.Set(resourceKeyServerOSImageID, server.SourceImageID)
	data.Set(resourceKeyServerMemoryGB, server.MemoryGB)
	data.Set(resourceKeyServerCPUCount, server.CPU.Count)

	disksByUnitID := getDisksByUnitID(server.Disks)

	// Match up image disks with server image disks.
	propertyHelper := propertyHelper(data)
	configuredImageDisks := propertyHelper.GetServerDisks(resourceKeyServerImageDisk)
	imageDisks := configuredImageDisks[:0]
	for _, imageDisk := range configuredImageDisks {
		disk, ok := disksByUnitID[imageDisk.SCSIUnitID]
		if !ok {
			continue
		}

		delete(disksByUnitID, imageDisk.SCSIUnitID)

		imageDisks = append(imageDisks, *disk)
	}
	propertyHelper.SetServerDisks(resourceKeyServerImageDisk, imageDisks)

	// TODO: Update additional disks.
	// Any disks remaining in disksByUnitID are, by process of elimination, additional disks.
	var additionalDisks []compute.VirtualMachineDisk
	for _, additionalDisk := range disksByUnitID {
		additionalDisks = append(additionalDisks, *additionalDisk)
	}
	propertyHelper.SetServerDisks(resourceKeyServerAdditionalDisk, additionalDisks)

	data.Set(resourceKeyServerPrimaryVLAN, *server.Network.PrimaryAdapter.VLANID)
	data.Set(resourceKeyServerPrimaryIPv4, *server.Network.PrimaryAdapter.PrivateIPv4Address)
	data.Set(resourceKeyServerPrimaryIPv6, *server.Network.PrimaryAdapter.PrivateIPv6Address)
	data.Set(resourceKeyServerNetworkDomainID, server.Network.NetworkDomainID)

	err = readServerTags(data, apiClient)
	if err != nil {
		return err
	}

	return nil
}

// Update a server resource.
func resourceServerUpdate(data *schema.ResourceData, provider interface{}) error {
	serverID := data.Id()

	// These changes can only be made through the V1 API (we're mostly using V2).
	// Later, we can come back and implement the required functionality in the compute API client.
	if data.HasChange(resourceKeyServerName) {
		return fmt.Errorf("Changing the 'name' property of a 'ddcloud_server' resource type is not yet implemented.")
	}

	if data.HasChange(resourceKeyServerDescription) {
		return fmt.Errorf("Changing the 'description' property of a 'ddcloud_server' resource type is not yet implemented.")
	}

	log.Printf("Update server '%s'.", serverID)

	apiClient := provider.(*providerState).Client()
	server, err := apiClient.GetServer(serverID)
	if err != nil {
		return err
	}

	data.Partial(true)

	propertyHelper := propertyHelper(data)

	var memoryGB, cpuCount *int
	if data.HasChange(resourceKeyServerMemoryGB) {
		memoryGB = propertyHelper.GetOptionalInt(resourceKeyServerMemoryGB, false)
	}
	if data.HasChange(resourceKeyServerCPUCount) {
		cpuCount = propertyHelper.GetOptionalInt(resourceKeyServerCPUCount, false)
	}

	if memoryGB != nil || cpuCount != nil {
		log.Printf("Server CPU / memory configuration change detected.")

		err = updateServerConfiguration(apiClient, server, memoryGB, cpuCount)
		if err != nil {
			return err
		}

		if data.HasChange(resourceKeyServerMemoryGB) {
			data.SetPartial(resourceKeyServerMemoryGB)
		}

		if data.HasChange(resourceKeyServerCPUCount) {
			data.SetPartial(resourceKeyServerCPUCount)
		}
	}

	if data.HasChange(resourceKeyServerImageDisk) {
		existingDisksByUnitID := getDisksByUnitID(server.Disks)
		err = updateImageDisks(existingDisksByUnitID, data, apiClient)
		if err != nil {
			return err
		}
	}

	// TODO: Handle additional disk changes.

	var primaryIPv4, primaryIPv6 *string
	if data.HasChange(resourceKeyServerPrimaryIPv4) {
		primaryIPv4 = propertyHelper.GetOptionalString(resourceKeyServerPrimaryIPv4, false)
	}
	if data.HasChange(resourceKeyServerPrimaryIPv6) {
		primaryIPv6 = propertyHelper.GetOptionalString(resourceKeyServerPrimaryIPv6, false)
	}

	if primaryIPv4 != nil || primaryIPv6 != nil {
		log.Printf("Server network configuration change detected.")

		err = updateServerIPAddress(apiClient, server, primaryIPv4, primaryIPv6)
		if err != nil {
			return err
		}

		if data.HasChange(resourceKeyServerPrimaryIPv4) {
			data.SetPartial(resourceKeyServerPrimaryIPv4)
		}

		if data.HasChange(resourceKeyServerPrimaryIPv6) {
			data.SetPartial(resourceKeyServerPrimaryIPv6)
		}
	}

	if data.HasChange(resourceKeyServerTag) {
		err = applyServerTags(data, apiClient)
		if err != nil {
			return err
		}

		data.SetPartial(resourceKeyServerTag)
	}

	data.Partial(false)

	return nil
}

// Delete a server resource.
func resourceServerDelete(data *schema.ResourceData, provider interface{}) error {
	var id, name, networkDomainID string

	id = data.Id()
	name = data.Get(resourceKeyServerName).(string)
	networkDomainID = data.Get(resourceKeyServerNetworkDomainID).(string)

	log.Printf("Delete server '%s' ('%s') in network domain '%s'.", id, name, networkDomainID)

	apiClient := provider.(*providerState).Client()
	server, err := apiClient.GetServer(id)
	if err != nil {
		return err
	}

	if server == nil {
		log.Printf("Server '%s' not found; will treat the server as having already been deleted.", id)

		return nil
	}

	if server.Started {
		log.Printf("Server '%s' is currently running. The server will be powered off.", id)

		err = apiClient.PowerOffServer(id)
		if err != nil {
			return err
		}

		_, err = apiClient.WaitForChange(compute.ResourceTypeServer, id, "Power off server", serverShutdownTimeout)
		if err != nil {
			return err
		}
	}

	log.Printf("Server '%s' is being deleted...", id)

	err = apiClient.DeleteServer(id)
	if err != nil {
		return err
	}

	return apiClient.WaitForDelete(compute.ResourceTypeServer, id, resourceDeleteTimeoutServer)
}

// updateServerConfiguration reconfigures a server, changing the allocated RAM and / or CPU count.
func updateServerConfiguration(apiClient *compute.Client, server *compute.Server, memoryGB *int, cpuCount *int) error {
	memoryDescription := "no change"
	if memoryGB != nil {
		memoryDescription = fmt.Sprintf("will change to %dGB", *memoryGB)
	}

	cpuCountDescription := "no change"
	if memoryGB != nil {
		memoryDescription = fmt.Sprintf("will change to %d", *cpuCount)
	}

	log.Printf("Update configuration for server '%s' (memory: %s, CPU: %s)...", server.ID, memoryDescription, cpuCountDescription)

	err := apiClient.ReconfigureServer(server.ID, memoryGB, cpuCount)
	if err != nil {
		return err
	}

	_, err = apiClient.WaitForChange(compute.ResourceTypeServer, server.ID, "Reconfigure server", resourceUpdateTimeoutServer)

	return err
}

// updateServerIPAddress notifies the compute infrastructure that a server's IP address has changed.
func updateServerIPAddress(apiClient *compute.Client, server *compute.Server, primaryIPv4 *string, primaryIPv6 *string) error {
	log.Printf("Update primary IP address(es) for server '%s'...", server.ID)

	primaryNetworkAdapterID := *server.Network.PrimaryAdapter.ID
	err := apiClient.NotifyServerIPAddressChange(primaryNetworkAdapterID, primaryIPv4, primaryIPv6)
	if err != nil {
		return err
	}

	compositeNetworkAdapterID := fmt.Sprintf("%s/%s", server.ID, primaryNetworkAdapterID)
	_, err = apiClient.WaitForChange(compute.ResourceTypeNetworkAdapter, compositeNetworkAdapterID, "Update adapter IP address", resourceUpdateTimeoutServer)

	return err
}

// When creating a server resource, synchronise the server's image disk attributes with its resource data
func createImageDisks(existingDisksByUnitID map[int]*compute.VirtualMachineDisk, data *schema.ResourceData, apiClient *compute.Client) error {
	propertyHelper := propertyHelper(data)

	serverID := data.Id()

	log.Printf("Create image disks for server '%s'...", serverID)

	imageDisks := propertyHelper.GetServerDisks(resourceKeyServerImageDisk)
	if len(imageDisks) == 0 {
		// Since this is the first time, populate image disks.
		var serverDisks []compute.VirtualMachineDisk
		for _, disk := range serverDisks {
			serverDisks = append(serverDisks, disk)
		}

		propertyHelper.SetServerDisks(resourceKeyServerImageDisk, serverDisks)
		propertyHelper.SetPartial(resourceKeyServerImageDisk)

		return nil
	}

	for _, imageDisk := range imageDisks {
		serverImageDisk, ok := existingDisksByUnitID[imageDisk.SCSIUnitID]
		if !ok {
			return fmt.Errorf("No disk was found with SCSI unit Id %d for server '%s'.", imageDisk.SCSIUnitID, serverID)
		}

		diskID := *serverImageDisk.ID

		if imageDisk.SizeGB == serverImageDisk.SizeGB {
			continue // Nothing to do.
		}

		if imageDisk.SizeGB < serverImageDisk.SizeGB {

			// Can't shrink disk, only grow it.
			return fmt.Errorf(
				"Cannot resize disk '%s' for server '%s' from %d to GB to %d (for now, disks can only be expanded).",
				diskID,
				serverID,
				serverImageDisk.SizeGB,
				imageDisk.SizeGB,
			)
		}

		// Do we need to expand the disk?
		if imageDisk.SizeGB > serverImageDisk.SizeGB {
			log.Printf(
				"Expanding disk '%s' for server '%s' (from %d GB to %d GB)...",
				diskID,
				serverID,
				serverImageDisk.SizeGB,
				imageDisk.SizeGB,
			)

			response, err := apiClient.ResizeServerDisk(serverID, diskID, imageDisk.SizeGB)
			if err != nil {
				return err
			}
			if response.Result != compute.ResultSuccess {
				return response.ToError("Unexpected result '%s' when resizing server disk '%s' for server '%s'.", response.Result, diskID, serverID)
			}

			resource, err := apiClient.WaitForChange(
				compute.ResourceTypeServer,
				serverID,
				"Resize disk",
				resourceUpdateTimeoutServer,
			)
			if err != nil {
				return err
			}

			server := resource.(*compute.Server)

			propertyHelper.SetServerDisks(resourceKeyServerImageDisk, server.Disks)
			propertyHelper.SetPartial(resourceKeyServerImageDisk)

			log.Printf(
				"Resized disk '%s' for server '%s' (from %d to GB to %d).",
				diskID,
				serverID,
				serverImageDisk.SizeGB,
				imageDisk.SizeGB,
			)

		}
	}

	return nil
}

// When updating a server resource, synchronise the server's image disk attributes with its resource data
func updateImageDisks(existingDisksByUnitID map[int]*compute.VirtualMachineDisk, data *schema.ResourceData, apiClient *compute.Client) error {
	propertyHelper := propertyHelper(data)

	serverID := data.Id()

	log.Printf("Update image disks for server '%s'...", serverID)

	imageDisks := propertyHelper.GetServerDisks(resourceKeyServerImageDisk)
	if len(imageDisks) == 0 {
		return fmt.Errorf("Invalid resource data for server '%s' (server has no image disks).", serverID)
	}

	for _, imageDisk := range imageDisks {
		serverImageDisk, ok := existingDisksByUnitID[imageDisk.SCSIUnitID]
		if !ok {
			return fmt.Errorf("No disk was found with SCSI unit Id %d for server '%s'.", imageDisk.SCSIUnitID, serverID)
		}

		diskID := *serverImageDisk.ID

		if imageDisk.SizeGB == serverImageDisk.SizeGB {
			continue // Nothing to do.
		}

		if imageDisk.SizeGB < serverImageDisk.SizeGB {
			// Can't shrink disk, only grow it.

			return fmt.Errorf(
				"Cannot shrink disk '%s' for server '%s' from %d to GB to %d (for now, disks can only be expanded).",
				diskID,
				serverID,
				serverImageDisk.SizeGB,
				imageDisk.SizeGB,
			)
		}

		// Do we need to expand the disk?
		if imageDisk.SizeGB > serverImageDisk.SizeGB {
			log.Printf(
				"Expanding disk '%s' for server '%s' (from %d GB to %d GB)...",
				diskID,
				serverID,
				serverImageDisk.SizeGB,
				imageDisk.SizeGB,
			)

			response, err := apiClient.ResizeServerDisk(serverID, diskID, imageDisk.SizeGB)
			if err != nil {
				return err
			}
			if response.Result != compute.ResultSuccess {
				return response.ToError("Unexpected result '%s' when resizing server disk '%s' for server '%s'.", response.Result, diskID, serverID)
			}

			resource, err := apiClient.WaitForChange(
				compute.ResourceTypeServer,
				serverID,
				"Resize disk",
				resourceUpdateTimeoutServer,
			)
			if err != nil {
				return err
			}

			server := resource.(*compute.Server)

			propertyHelper.SetServerDisks(resourceKeyServerImageDisk, server.Disks)
			propertyHelper.SetPartial(resourceKeyServerImageDisk)

			log.Printf(
				"Resized disk '%s' for server '%s' (from %d to GB to %d).",
				diskID,
				serverID,
				serverImageDisk.SizeGB,
				imageDisk.SizeGB,
			)

		}
	}

	return nil
}

func getDisksByUnitID(disks []compute.VirtualMachineDisk) map[int]*compute.VirtualMachineDisk {
	disksByUnitID := make(map[int]*compute.VirtualMachineDisk)
	for _, disk := range disks {
		disksByUnitID[disk.SCSIUnitID] = &disk
	}

	return disksByUnitID
}

func hashDiskUnitID(item interface{}) int {
	disk, ok := item.(compute.VirtualMachineDisk)
	if ok {
		return disk.SCSIUnitID
	}

	diskData := item.(map[string]interface{})

	return diskData[resourceKeyServerDiskUnitID].(int)
}

func hashServerTag(item interface{}) int {
	tagData := item.(map[string]interface{})

	return schema.HashString(
		tagData[resourceKeyServerTagName].(string),
	)
}

// Apply configured tags to a server.
func applyServerTags(data *schema.ResourceData, apiClient *compute.Client) error {
	propertyHelper := propertyHelper(data)

	serverID := data.Id()

	log.Printf("Configuring tags for server '%s'...", serverID)

	configuredTags := propertyHelper.GetTags(resourceKeyServerTag)

	// TODO: Support multiple pages of results.
	serverTags, err := apiClient.GetAssetTags(serverID, compute.AssetTypeServer, nil)
	if err != nil {
		return err
	}

	// Capture any tags that are no-longer needed.
	unusedTags := &schema.Set{
		F: schema.HashString,
	}
	for _, tag := range serverTags.Items {
		unusedTags.Add(tag.Name)
	}
	for _, tag := range configuredTags {
		unusedTags.Remove(tag.Name)
	}

	log.Printf("Applying %d tags to server '%s'...", len(configuredTags), serverID)

	response, err := apiClient.ApplyAssetTags(serverID, compute.AssetTypeServer, configuredTags...)
	if err != nil {
		return err
	}

	if response.ResponseCode != compute.ResponseCodeOK {
		return response.ToError("Failed to apply %d tags to server '%s' (response code '%s'): %s", len(configuredTags), serverID, response.ResponseCode, response.Message)
	}

	// Trim unused tags (currently-configured tags will overwrite any existing values).
	if unusedTags.Len() > 0 {
		unusedTagNames := make([]string, unusedTags.Len())
		for index, unusedTagName := range unusedTags.List() {
			unusedTagNames[index] = unusedTagName.(string)
		}

		log.Printf("Removing %d unused tags from server '%s'...", len(unusedTagNames), serverID)

		response, err = apiClient.RemoveAssetTags(serverID, compute.AssetTypeServer, unusedTagNames...)
		if err != nil {
			return err
		}

		if response.ResponseCode != compute.ResponseCodeOK {
			return response.ToError("Failed to remove %d tags from server '%s' (response code '%s'): %s", len(configuredTags), serverID, response.ResponseCode, response.Message)
		}
	}

	return nil
}

// Read tags from a server and update resource data accordingly.
func readServerTags(data *schema.ResourceData, apiClient *compute.Client) error {
	propertyHelper := propertyHelper(data)

	serverID := data.Id()

	log.Printf("Reading tags for server '%s'...", serverID)

	result, err := apiClient.GetAssetTags(serverID, compute.AssetTypeServer, nil)
	if err != nil {
		return err
	}

	log.Printf("Read %d tags for server '%s'.", result.PageCount, serverID)

	// TODO: Handle multiple pages of results.

	tags := make([]compute.Tag, len(result.Items))
	for index, tagDetail := range result.Items {
		tags[index] = compute.Tag{
			Name:  tagDetail.Name,
			Value: tagDetail.Value,
		}
	}

	propertyHelper.SetTags(resourceKeyServerTag, tags)

	return nil
}

func schemaServerDisk() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		Default:  nil,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				resourceKeyServerDiskID: &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
				},
				resourceKeyServerDiskSizeGB: &schema.Schema{
					Type:     schema.TypeInt,
					Required: true,
				},
				resourceKeyServerDiskUnitID: &schema.Schema{
					Type:     schema.TypeInt,
					Required: true,
				},
				resourceKeyServerDiskSpeed: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "STANDARD",
				},
			},
		},
		Set: hashDiskUnitID,
	}
}
