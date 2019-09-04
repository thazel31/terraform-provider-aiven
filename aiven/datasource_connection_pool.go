// Copyright (c) 2019 Aiven, Helsinki, Finland. https://aiven.io/
package aiven

import (
	"github.com/thazel31/aiven-go-client"
	"github.com/hashicorp/terraform/helper/schema"
)

func datasourceConnectionPool() *schema.Resource {
	return &schema.Resource{
		Read:   datasourceConnectionPoolRead,
		Schema: resourceSchemaAsDatasourceSchema(aivenConnectionPoolSchema, "project", "service_name", "pool_name"),
	}
}

func datasourceConnectionPoolRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*aiven.Client)

	projectName := d.Get("project").(string)
	serviceName := d.Get("service_name").(string)
	poolName := d.Get("pool_name").(string)

	pool, err := client.ConnectionPools.Get(projectName, serviceName, poolName)
	if err != nil {
		return err
	}

	d.SetId(buildResourceID(projectName, serviceName, poolName))
	return copyConnectionPoolPropertiesFromAPIResponseToTerraform(d, pool, projectName, serviceName)
}
