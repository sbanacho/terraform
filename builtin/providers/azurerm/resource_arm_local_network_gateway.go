package azurerm

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/core/http"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmLocalNetworkGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLocalNetworkGatewayCreate,
		Read:   resourceArmLocalNetworkGatewayRead,
		Update: resourceArmLocalNetworkGatewayCreate,
		Delete: resourceArmLocalNetworkGatewayDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"gateway_address": {
				Type:     schema.TypeString,
				Required: true,
			},

			"address_space": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceArmLocalNetworkGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	lnetClient := meta.(*ArmClient).localNetConnClient

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	ipAddress := d.Get("gateway_address").(string)

	// fetch the 'address_space_prefixes:
	prefixes := []string{}
	for _, pref := range d.Get("address_space").([]interface{}) {
		prefixes = append(prefixes, pref.(string))
	}

	gateway := network.LocalNetworkGateway{
		Name:     &name,
		Location: &location,
		Properties: &network.LocalNetworkGatewayPropertiesFormat{
			LocalNetworkAddressSpace: &network.AddressSpace{
				AddressPrefixes: &prefixes,
			},
			GatewayIPAddress: &ipAddress,
		},
	}

	_, err := lnetClient.CreateOrUpdate(resGroup, name, gateway, make(chan struct{}))
	if err != nil {
		return fmt.Errorf("Error creating Azure ARM Local Network Gateway '%s': %s", name, err)
	}

	read, err := lnetClient.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Virtual Network %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmLocalNetworkGatewayRead(d, meta)
}

// resourceArmLocalNetworkGatewayRead goes ahead and reads the state of the corresponding ARM local network gateway.
func resourceArmLocalNetworkGatewayRead(d *schema.ResourceData, meta interface{}) error {
	lnetClient := meta.(*ArmClient).localNetConnClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["localNetworkGateways"]
	resGroup := id.ResourceGroup

	resp, err := lnetClient.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading the state of Azure ARM local network gateway '%s': %s", name, err)
	}

	d.Set("name", resp.Name)
	d.Set("location", resp.Location)
	d.Set("gateway_address", resp.Properties.GatewayIPAddress)

	prefs := []string{}
	if ps := *resp.Properties.LocalNetworkAddressSpace.AddressPrefixes; ps != nil {
		prefs = ps
	}
	d.Set("address_space", prefs)

	return nil
}

// resourceArmLocalNetworkGatewayDelete deletes the specified ARM local network gateway.
func resourceArmLocalNetworkGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	lnetClient := meta.(*ArmClient).localNetConnClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["localNetworkGateways"]
	resGroup := id.ResourceGroup

	_, err = lnetClient.Delete(resGroup, name, make(chan struct{}))
	if err != nil {
		return fmt.Errorf("Error issuing Azure ARM delete request of local network gateway '%s': %s", name, err)
	}

	return nil
}
