package kafkasub

/*
This is the Kafka server setup to support these tests:

sasl.enabled.mechanisms=PLAIN
sasl.mechanism.inter.broker.protocol=PLAIN
advertised.listeners=PLAINTEXT://bilbo:9092,SSL://bilbo:9093,SASL_PLAINTEXT://bilbo:9094,SASL_SSL://bilbo:9095

ssl.keystore.location=/local/opt/kafka/kafka_2.11-0.10.2.0/keys/kafka.jks
ssl.keystore.password=sauron
ssl.key.password=sauron
ssl.truststore.location=/local/opt/kafka/kafka_2.11-0.10.2.0/keys/kafka.jks
ssl.truststore.password=sauron
ssl.client.auth=none
ssl.enabled.protocols=TLSv1.2,TLSv1.1,TLSv1


*/
import (
	"context"
	"encoding/json"
	"fmt"
	"os/signal"
	"syscall"
	"testing"

	"io/ioutil"

	"time"

	"log"
	"os"

	"github.com/TIBCOSoftware/flogo-lib/core/action"
	"github.com/TIBCOSoftware/flogo-lib/core/trigger"
)

var jsonMetadata = getJsonMetadata()
var listentime time.Duration = 10

func getJsonMetadata() string {
	jsonMetadataBytes, err := ioutil.ReadFile("trigger.json")
	if err != nil {
		panic("No Json Metadata found for trigger.json path")
	}
	return string(jsonMetadataBytes)
}

const testConfig string = `{
  "name": "tibco-kafkasub",
  "settings": {
    "BrokerUrl": "192.168.10.69:9092"
  },
  "handlers": [
    {
      "actionId": "kafka_message",
      "settings": {
        "Topic": "syslog"
      }
    }
  ],
	"outputs": [
    {
      "name": "message",
      "type": "string"
    }
  ]
}`

type TestRunner struct {
}

// Run implements action.Runner.Run
func (tr *TestRunner) Run(context context.Context, action action.Action, uri string, options interface{}) (code int, data interface{}, err error) {
	log.Printf("Ran Action: %v", uri)
	return 0, nil, nil
}

func consoleHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received console interrupt.  Exiting.")
		time.Sleep(time.Second * 3)
		os.Exit(1)
	}()
}

func runTest(config *trigger.Config, expectSucceed bool, testName string, configOnly bool) error {
	log.Printf("Test %s starting\n", testName)
	defer func() error {
		if r := recover(); r != nil {
			if expectSucceed {
				log.Println("Test %s was expected to succeed but did not because: ", testName, r)
				return fmt.Errorf("%s", r)
			}
		}
		return nil
	}()
	f := &KafkasubFactory{}
	tgr := f.New(config)
	log.Printf("\t%s trigger created\n", testName)
	runner := &TestRunner{}
	tgr.Init(runner)
	log.Printf("\t%s trigger initialized \n", testName)
	if configOnly {
		log.Printf("Test %s complete\n", testName)
		return nil
	}
	defer tgr.Stop()
	tgr.Start()
	log.Printf("\t%s listening for messages for %d seconds\n", testName, listentime)
	time.Sleep(time.Second * listentime)
	log.Printf("Test %s complete\n", testName)
	return nil
}
func TestInit(t *testing.T) {
	consoleHandler()
	config := trigger.Config{}
	error := json.Unmarshal([]byte(testConfig), &config)
	if error != nil {
		log.Printf("Failed to unmarshal the config args:%s", error)
		t.Fail()
	}
	runTest(&config, true, "TestInit", true)
	config.Settings["BrokerUrl"] = "192.168.10.1:9092,127.0.0.1:9092,a.b.c.c:9093,a.123.z-fr.c:9096"
	runTest(&config, true, "TestInit", true)

}

func TestEndpoint(t *testing.T) {
	config := trigger.Config{}
	error := json.Unmarshal([]byte(testConfig), &config)
	if error != nil {
		log.Printf("Failed to unmarshal the config args:%s", error)
		t.Fail()
	}
	runTest(&config, true, "TestEndPoint", false)
}

func TestMultiBrokers(t *testing.T) {
	config := trigger.Config{}
	json.Unmarshal([]byte(testConfig), &config)
	config.Settings["BrokerUrl"] = "bilbo:9092,bilbo:9092,bilbo:9092"
	runTest(&config, true, "TestMultiBrokers", false)
}

func TestTLS(t *testing.T) {
	config := trigger.Config{}
	json.Unmarshal([]byte(testConfig), &config)
	config.Handlers[0].Settings["truststore"] = "/opt/kafka/kafka_2.11-0.10.2.0/keys/trust"
	config.Settings["BrokerUrl"] = "bilbo:9093"
	runTest(&config, true, "TestTLS", false)
}

func TestSASL(t *testing.T) {
	config := trigger.Config{}
	json.Unmarshal([]byte(testConfig), &config)
	config.Handlers[0].Settings["user"] = "wcn00"
	config.Handlers[0].Settings["password"] = "sauron"
	config.Settings["BrokerUrl"] = "bilbo.wcn.org:9094"
	runTest(&config, true, "TestSASL", false)
}

func TestSASL_TLS(t *testing.T) {
	config := trigger.Config{}
	json.Unmarshal([]byte(testConfig), &config)
	config.Handlers[0].Settings["truststore"] = "/opt/kafka/kafka_2.11-0.10.2.0/keys/trust"
	config.Handlers[0].Settings["user"] = "wcn00"
	config.Handlers[0].Settings["password"] = "sauron"
	config.Settings["BrokerUrl"] = "bilbo:9095"
	runTest(&config, true, "TestSASL_TLS", false)
}

func TestNumericIpaddr(t *testing.T) {
	config := trigger.Config{}
	json.Unmarshal([]byte(testConfig), &config)
	config.Settings["BrokerUrl"] = "192.168.10.69:9092"
	runTest(&config, true, "TestNumericIpaddr", false)
}
func TestFailingEndpoint(t *testing.T) {
	config := trigger.Config{}
	json.Unmarshal([]byte(testConfig), &config)
	config.Handlers[0].Settings["partitions"] = "21,31" //negative test!!!
	defer func() {
		if r := recover(); r != nil {
			log.Println("Test TestFailingEndpoint failed as expected.")
		}
	}()
	log.Println("This test is expected to fail!!!!!")
	runTest(&config, false, "TestFailingEndpoint", false)
}
