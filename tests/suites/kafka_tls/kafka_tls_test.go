package kafka_tls

import (
	"fmt"
	"testing"

	"github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/mesosphere/kudo-kafka-operator/tests/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var (
	customNamespace = "kafka-tls-test"
)

var _ = Describe("KafkaTLS", func() {
	Describe("[Kafka TLS]", func() {
		Context("tls enabled", func() {
			kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
				Namespace: utils.String(customNamespace),
			})
			It("statefulset should have 1 replica with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(suites.DefaultZkStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(suites.DefaultKafkaStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(suites.DefaultKafkaStatefulSetName, customNamespace)).To(Equal(1))
			})
			It("verify the certs", func() {
				output, err := kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", suites.DefaultContainerName,
					[]string{"findmnt", "/etc/tls/bin"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("kubernetes.io~configmap/enable-tls"))

				crtModulus, err := kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", suites.DefaultContainerName,
					[]string{"openssl", "x509", "-noout", "-modulus", "-in", "/etc/tls/certs/tls.crt"})
				Expect(err).To(BeNil())
				keyModulus, err := kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", suites.DefaultContainerName,
					[]string{"openssl", "rsa", "-noout", "-modulus", "-in", "/etc/tls/certs/tls.key"})

				Expect(err).To(BeNil())
				Expect(crtModulus).To(Equal(keyModulus))
			})
			It("verify the SSL listener", func() {
				output, err := kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", suites.DefaultContainerName,
					[]string{"grep", "ListenerName", "/var/lib/kafka/data/server.log"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("ListenerName(INTERNAL),SSL"))
			})
		})
	})
})

var _ = BeforeSuite(func() {
	utils.TearDown(customNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	utils.KClient.CreateNamespace(customNamespace, false)
	utils.KClient.CreateTLSCertSecret(customNamespace, "kafka-tls", "Kafka")
	utils.InstallKudoOperator(customNamespace, utils.ZK_INSTANCE, utils.ZK_FRAMEWORK_DIR_ENV, map[string]string{
		"MEMORY":     "256Mi",
		"CPUS":       "0.25",
		"NODE_COUNT": "1",
	})
	utils.KClient.WaitForStatefulSetCount(suites.DefaultZkStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWaitSeconds)
	utils.InstallKudoOperator(customNamespace, utils.KAFKA_INSTANCE, utils.KAFKA_FRAMEWORK_DIR_ENV, map[string]string{
		"BROKER_MEM":                       "1Gi",
		"BROKER_CPUS":                      "0.25",
		"BROKER_COUNT":                     "1",
		"TLS_SECRET_NAME":                  "kafka-tls",
		"TRANSPORT_ENCRYPTION_ENABLED":     "true",
		"ZOOKEEPER_URI":                    "zookeeper-instance-zookeeper-0.zookeeper-instance-hs:2181",
		"OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
	})
	utils.KClient.WaitForStatefulSetCount(suites.DefaultKafkaStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWaitSeconds)
})

var _ = AfterSuite(func() {
	utils.TearDown(customNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	utils.KClient.DeleteNamespace(customNamespace)
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-tls"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaTLS Suite", []Reporter{junitReporter})
}
