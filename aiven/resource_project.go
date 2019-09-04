// Copyright (c) 2017 jelmersnoeck
// Copyright (c) 2018 Aiven, Helsinki, Finland. https://aiven.io/
package aiven

import (
	"github.com/thazel31/aiven-go-client"
	"github.com/hashicorp/terraform/helper/schema"
)

var aivenProjectSchema = map[string]*schema.Schema{
	"billing_address": {
		Type:        schema.TypeString,
		Description: "Billing name and address of the project",
		Optional:    true,
	},
	"billing_emails": {
		Type:        schema.TypeSet,
		Description: "Billing contact emails of the project",
		Elem:        &schema.Schema{Type: schema.TypeString},
		Optional:    true,
	},
	"ca_cert": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Project root CA. This is used by some services like Kafka to sign service certificate",
		Optional:    true,
	},
	"card_id": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Credit card ID",
	},
	"copy_from_project": {
		Type:             schema.TypeString,
		Optional:         true,
		Description:      "Copy properties from another project. Only has effect when a new project is created.",
		DiffSuppressFunc: createOnlyDiffSuppressFunc,
	},
	"country_code": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Billing country code of the project",
	},
	"project": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "Project name",
	},
	"technical_emails": {
		Type:        schema.TypeSet,
		Description: "Technical contact emails of the project",
		Elem:        &schema.Schema{Type: schema.TypeString},
		Optional:    true,
	},
	"vat_id": {
		Type:        schema.TypeString,
		Description: "EU VAT identification number",
		Optional:    true,
	},
}

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectDelete,
		Exists: resourceProjectExists,
		Importer: &schema.ResourceImporter{
			State: resourceProjectState,
		},

		Schema: aivenProjectSchema,
	}
}

func resourceProjectCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*aiven.Client)
	cardID, err := getLongCardID(client, d.Get("card_id").(string))
	if err != nil {
		return err
	}
	projectName := d.Get("project").(string)
	vatID := d.Get("vat_id").(string)
	project, err := client.Projects.Create(
		aiven.CreateProjectRequest{
			BillingAddress:  optionalStringPointer(d, "billing_address"),
			BillingEmails:   contactEmailListForAPI(d, "billing_emails", true),
			CardID:          cardID,
			CopyFromProject: d.Get("copy_from_project").(string),
			CountryCode:     optionalStringPointer(d, "country_code"),
			Project:         projectName,
			TechnicalEmails: contactEmailListForAPI(d, "technical_emails", true),
			VatID: vatID,
		},
	)
	if err != nil {
		return err
	}

	d.SetId(projectName)
	resourceProjectGetCACert(project.Name, client, d)
	return nil
}

func resourceProjectRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*aiven.Client)

	project, err := client.Projects.Get(d.Id())
	if err != nil {
		return err
	}

	// Don't set card id unconditionally to prevent converting short card id format to long
	currentCardID, err := getLongCardID(client, d.Get("card_id").(string))
	if err != nil || currentCardID != project.Card.CardID {
		d.Set("card_id", project.Card.CardID)
	}
	setProjectTerraformProperties(d, client, project)
	return nil
}

func resourceProjectUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*aiven.Client)

	cardID, err := getLongCardID(client, d.Get("card_id").(string))
	if err != nil {
		return err
	}
	billingAddress := d.Get("billing_address").(string)
	countryCode := d.Get("country_code").(string)
	project, err := client.Projects.Update(
		d.Get("project").(string),
		aiven.UpdateProjectRequest{
			BillingAddress:  &billingAddress,
			BillingEmails:   contactEmailListForAPI(d, "billing_emails", false),
			CardID:          cardID,
			CountryCode:     &countryCode,
			TechnicalEmails: contactEmailListForAPI(d, "technical_emails", false),
		},
	)
	if err != nil {
		return err
	}

	d.SetId(project.Name)
	return nil
}

func resourceProjectDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*aiven.Client)

	return client.Projects.Delete(d.Id())
}

func resourceProjectExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := m.(*aiven.Client)

	_, err := client.Projects.Get(d.Get("project").(string))
	return resourceExists(err)
}

func resourceProjectState(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client := m.(*aiven.Client)

	project, err := client.Projects.Get(d.Id())
	if err != nil {
		return nil, err
	}

	d.Set("card_id", project.Card.CardID)
	setProjectTerraformProperties(d, client, project)

	return []*schema.ResourceData{d}, nil
}

func resourceProjectGetCACert(project string, client *aiven.Client, d *schema.ResourceData) {
	ca, err := client.CA.Get(project)
	if err == nil {
		d.Set("ca_cert", ca)
	}
}

func getLongCardID(client *aiven.Client, cardID string) (string, error) {
	card, err := client.CardsHandler.Get(cardID)
	if err != nil {
		return "", err
	}
	if card != nil {
		return card.CardID, nil
	}
	return cardID, nil
}

func contactEmailListForAPI(d *schema.ResourceData, field string, newResource bool) *[]*aiven.ContactEmail {
	var results []*aiven.ContactEmail
	// We don't want to send empty list for new resource if data is copied from other
	// project to prevent accidental override of the emails being copied. Empty array
	// should be sent if user has explicitly defined that even when copy_from_project
	// is set but Terraform does not support checking that; d.GetOkExists returns false
	// even if the value is set (to empty).
	if len(d.Get("copy_from_project").(string)) == 0 || !newResource {
		results = []*aiven.ContactEmail{}
	}
	valuesInterface, ok := d.GetOk(field)
	if ok && valuesInterface != nil {
		for _, emailInterface := range valuesInterface.(*schema.Set).List() {
			results = append(results, &aiven.ContactEmail{Email: emailInterface.(string)})
		}
	}
	if results == nil {
		return nil
	}
	return &results
}

func contactEmailListForTerraform(d *schema.ResourceData, field string, contactEmails []*aiven.ContactEmail) {
	_, existsBefore := d.GetOk(field)
	if !existsBefore && len(contactEmails) == 0 {
		return
	}
	results := []string{}
	for _, contactEmail := range contactEmails {
		results = append(results, contactEmail.Email)
	}
	d.Set(field, results)
}

func setProjectTerraformProperties(d *schema.ResourceData, client *aiven.Client, project *aiven.Project) {
	d.Set("billing_address", project.BillingAddress)
	contactEmailListForTerraform(d, "billing_emails", project.BillingEmails)
	d.Set("country_code", project.CountryCode)
	d.Set("project", project.Name)
	contactEmailListForTerraform(d, "technical_emails", project.TechnicalEmails)
	resourceProjectGetCACert(project.Name, client, d)
}
