package polaris_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
)

var _ = Describe("WorkloadBuilder", func() {
	var (
		ctx         context.Context
		diApp       *fxtest.App
		appStore    bkmsapp.ApplicationStore
		envService  *env.EnvService
		store       polaris.PolarisConfigStore
		builder     *polaris.WorkloadBuilder
		app         *bkmsapp.Application
		environment *envmodel.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			env.FxModule,
			polaris.FxModule,
			fx.Populate(&appStore, &envService, &store, &builder),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		environment = dbfactory.Env(ctx, envService, app.WorkspaceID)
	})

	AfterEach(func() {
		Expect(store.DeleteByApp(ctx, app.ID)).To(Succeed())
		diApp.RequireStop()
	})

	createConfig := func(name string, scopeEnvNames []string, port int32, labels map[string]string) *polaris.PolarisConfig {
		config := &polaris.PolarisConfig{
			Name:  name,
			AppID: app.ID,
			Properties: polaris.Properties{
				InstanceKey:       name,
				PolarisName:       "${{env.POLARIS_NAME}}",
				PolarisNamespace:  "Production",
				PolarisToken:      "${{env.POLARIS_TOKEN}}",
				ServicePort:       port,
				Direct:            true,
				KeepNotReadyPod:   true,
				EnableHealthCheck: true,
				ServiceLabels:     labels,
			},
			ScopeEnvNames: scopeEnvNames,
		}
		Expect(store.Create(ctx, config)).To(Succeed())
		return config
	}

	It("builds Polaris resources and injects the matching service port", func() {
		config := createConfig("primary", []string{environment.Name}, 8080, map[string]string{
			"environment": "${{env.ENV_NAME}}",
			"team":        "platform",
		})
		podSpec := corev1.PodSpec{Containers: []corev1.Container{
			{Name: "sidecar"},
			{Name: "main"},
		}}

		result, err := builder.Build(ctx, app, environment,
			map[string]string{"ENV_NAME": environment.Name, "POLARIS_NAME": "rendered-name"},
			podSpec,
			"main",
			nil,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.ExtraObjects).To(HaveLen(2))
		Expect(podSpec.Containers[1].Ports).To(BeEmpty())
		Expect(result.PodSpec.Containers[0].Ports).To(BeEmpty())
		Expect(result.PodSpec.Containers[1].Ports).To(ConsistOf(corev1.ContainerPort{
			Name:          "polaris-8080",
			ContainerPort: 8080,
			Protocol:      corev1.ProtocolTCP,
		}))

		objectsByKind := make(map[string]unstructured.Unstructured, len(result.ExtraObjects))
		for _, object := range result.ExtraObjects {
			objectsByKind[object.GetKind()] = object
		}
		Expect(objectsByKind).To(HaveKey("PolarisConfig"))
		Expect(objectsByKind).To(HaveKey("Service"))

		expectedBaseName := strings.ToLower(app.Name + "-" + config.Name)
		cr := objectsByKind["PolarisConfig"]
		Expect(cr.GetAPIVersion()).To(Equal("tkex.tencent.com/v1"))
		Expect(cr.GetName()).To(Equal(expectedBaseName + "-polaris"))
		Expect(nestedString(cr.Object, "spec", "polaris", "name")).To(Equal("${{env.POLARIS_NAME}}"))
		Expect(nestedString(cr.Object, "spec", "polaris", "token")).To(Equal("${{env.POLARIS_TOKEN}}"))

		services, found, err := unstructured.NestedSlice(cr.Object, "spec", "services")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(services).To(HaveLen(1))
		serviceSpec, ok := services[0].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(serviceSpec["name"]).To(Equal(expectedBaseName + "-polaris-service"))
		Expect(serviceSpec["namespace"]).To(Equal(environment.Cluster.Namespace))
		Expect(serviceSpec["port"]).To(BeEquivalentTo(8080))
		Expect(serviceSpec["direct"]).To(BeTrue())
		Expect(serviceSpec["keepNotReadyPod"]).To(BeTrue())
		Expect(serviceSpec["enableHealthCheck"]).To(BeTrue())
		Expect(serviceSpec["weight"]).To(BeEquivalentTo(polaris.DefaultEnvWeight))
		extraMeta, ok := serviceSpec["extraMeta"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(extraMeta).To(Equal(map[string]any{
			"environment": environment.Name,
			"team":        "platform",
		}))

		service := objectsByKind["Service"]
		Expect(service.GetName()).To(Equal(expectedBaseName + "-polaris-service"))
		Expect(nestedString(service.Object, "spec", "selector", "app.kubernetes.io/name")).To(Equal(app.Name))
		ports, found, err := unstructured.NestedSlice(service.Object, "spec", "ports")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(ports).To(HaveLen(1))
		Expect(ports[0].(map[string]any)["port"]).To(BeEquivalentTo(8080))
	})

	It("uses the environment weight in the PolarisConfig resource", func() {
		config := createConfig("env-weight", []string{environment.Name}, 8080, nil)
		Expect(store.UpsertEnvWeight(ctx, app.ID, config.Name, environment.Name, 35)).To(Succeed())

		result, err := builder.Build(
			ctx,
			app,
			environment,
			nil,
			corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			"main",
			nil,
		)
		Expect(err).NotTo(HaveOccurred())

		var polarisConfig unstructured.Unstructured
		for _, object := range result.ExtraObjects {
			if object.GetKind() == "PolarisConfig" {
				polarisConfig = object
				break
			}
		}
		Expect(polarisConfig.Object).NotTo(BeNil())
		services, found, err := unstructured.NestedSlice(polarisConfig.Object, "spec", "services")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(services[0].(map[string]any)["weight"]).To(BeEquivalentTo(int32(35)))
	})

	It("filters configs by environment and replaces a conflicting container port", func() {
		createConfig("matching", []string{environment.Name}, 8080, nil)
		createConfig("other", []string{"another-env"}, 9090, nil)
		result, err := builder.Build(ctx, app, environment,
			map[string]string{},
			corev1.PodSpec{Containers: []corev1.Container{{
				Name: "main",
				Ports: []corev1.ContainerPort{{
					Name:          "existing",
					ContainerPort: 8080,
					Protocol:      corev1.ProtocolUDP,
				}},
			}}},
			"main",
			nil,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.ExtraObjects).To(HaveLen(2))
		Expect(result.PodSpec.Containers[0].Ports).To(Equal([]corev1.ContainerPort{{
			Name:          "polaris-8080",
			ContainerPort: 8080,
			Protocol:      corev1.ProtocolTCP,
		}}))
	})

	It("returns the workload unchanged when no config matches the environment", func() {
		createConfig("other", []string{"another-env"}, 9090, nil)
		podSpec := corev1.PodSpec{Containers: []corev1.Container{{
			Name:  "main",
			Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 80}},
		}}}

		result, err := builder.Build(ctx, app, environment, nil, podSpec, "main", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.ExtraObjects).To(BeEmpty())
		Expect(result.PodSpec).To(Equal(podSpec))
	})

	It("returns an error for an invalid service label template", func() {
		createConfig("invalid-label", []string{environment.Name}, 8080, map[string]string{
			"invalid": "${{env.UNFINISHED",
		})

		_, err := builder.Build(
			ctx, app, environment, nil,
			corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			"main",
			nil,
		)
		Expect(err).To(MatchError(ContainSubstring("render service label invalid")))
	})
})

func nestedString(object map[string]any, fields ...string) string {
	value, found, err := unstructured.NestedString(object, fields...)
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())
	return value
}
