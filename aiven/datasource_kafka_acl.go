// Copyright (c) 2019 Aiven, Helsinki, Finland. https://aiven.io/
package aiven

import (
	"fmt"

	"github.com/thazel31/aiven-go-client"
	"github.com/hashicorp/terraform/helper/schema"
)

func datasourceKafkaACL() *schema.Resource {
	return &schema.Resource{
		Read:   datasourceKafkaACLRead,
		Schema: resourceSchemaAsDatasourceSchema(aivenKafkaACLSchema, "project", "service_name", "topic", "username"),
	}
}

func datasourceKafkaACLRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*aiven.Client)

	projectName := d.Get("project").(string)
	serviceName := d.Get("service_name").(string)
	topic := d.Get("topic").(string)
	userName := d.Get("username").(string)

	acls, err := client.KafkaACLs.List(projectName, serviceName)
	if err != nil {
		return err
	}

	for _, acl := range acls {
		if acl.Topic == topic && acl.Username == userName {
			d.SetId(buildResourceID(projectName, serviceName, acl.ID))
			return copyKafkaACLPropertiesFromAPIResponseToTerraform(d, acl, projectName, serviceName)
		}
	}

	return fmt.Errorf("KafkaACL %s/%s not found", topic, userName)
}
