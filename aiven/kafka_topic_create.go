package aiven

import (
	"log"
	"strings"
	"time"

	"github.com/thazel31/aiven-go-client"
	"github.com/hashicorp/terraform/helper/resource"
)

// KafkaTopicCreateWaiter is used to create topics. Since topics are often
// created right after Kafka service is created there may be temporary issues
// that prevent creating the topics like all brokers not being online. This
// allows retrying the operation until failing it.
type KafkaTopicCreateWaiter struct {
	Client        *aiven.Client
	Project       string
	ServiceName   string
	CreateRequest aiven.CreateKafkaTopicRequest
}

// RefreshFunc will call the Aiven client and refresh it's state.
func (w *KafkaTopicCreateWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		err := w.Client.KafkaTopics.Create(
			w.Project,
			w.ServiceName,
			w.CreateRequest,
		)

		if err != nil {
			// If some brokers are offline while the request is being executed
			// the operation may fail.
			aivenError, ok := err.(aiven.Error)
			if ok && aivenError.Status == 409 && !strings.Contains(aivenError.Message, "already exists") {
				log.Printf("[DEBUG] Got error %v while waiting for topic to be created.", aivenError)
				return nil, "CREATING", nil
			}
			return nil, "", err
		}

		return w.CreateRequest.TopicName, "CREATED", nil
	}
}

// Conf sets up the configuration to refresh.
func (w *KafkaTopicCreateWaiter) Conf() *resource.StateChangeConf {
	state := &resource.StateChangeConf{
		Pending: []string{"CREATING"},
		Target:  []string{"CREATED"},
		Refresh: w.RefreshFunc(),
	}
	state.Delay = 10 * time.Second
	state.Timeout = 1 * time.Minute
	state.MinTimeout = 2 * time.Second
	return state
}
