package completion

import (
	"reflect"
	"sort"
	"testing"

	"github.com/openshift/odo/pkg/component"
	"github.com/openshift/odo/pkg/testingutil"
	"github.com/posener/complete"

	scv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	appsv1 "github.com/openshift/api/apps/v1"
	applabels "github.com/openshift/odo/pkg/application/labels"
	componentlabels "github.com/openshift/odo/pkg/component/labels"
	"github.com/openshift/odo/pkg/occlient"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
)

func TestServicePlanCompletionHandler(t *testing.T) {
	serviceClassList := &scv1beta1.ClusterServiceClassList{
		Items: []scv1beta1.ClusterServiceClass{testingutil.FakeClusterServiceClass("class name", "dummy")},
	}
	tests := []struct {
		name                 string
		returnedServiceClass *scv1beta1.ClusterServiceClassList
		returnedServicePlan  []scv1beta1.ClusterServicePlan
		output               []string
		parsedArgs           parsedArgs
	}{
		{
			name: "Case 0: no service name supplied",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create"},
				},
			},
			output: []string{},
		},
		{
			name:                 "Case 1: single plan exists",
			returnedServiceClass: serviceClassList,
			returnedServicePlan:  []scv1beta1.ClusterServicePlan{testingutil.FakeClusterServicePlan("default", 1)},
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create", "class name"},
				},
			},
			output: []string{"default"},
		},
		{
			name:                 "Case 2: multiple plans exist",
			returnedServiceClass: serviceClassList,
			returnedServicePlan: []scv1beta1.ClusterServicePlan{
				testingutil.FakeClusterServicePlan("plan1", 1),
				testingutil.FakeClusterServicePlan("plan2", 2),
			},
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create", "class name"},
				},
			},
			output: []string{"plan1", "plan2"},
		},
	}

	for _, tt := range tests {
		client, fakeClientSet := occlient.FakeNew()
		context := genericclioptions.NewFakeContext("project", "app", "component", client)

		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "clusterserviceclasses", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, tt.returnedServiceClass, nil
		})

		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "clusterserviceplans", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &scv1beta1.ClusterServicePlanList{Items: tt.returnedServicePlan}, nil
		})

		completions := ServicePlanCompletionHandler(nil, tt.parsedArgs, context)

		// Sort the output and expected output in order to avoid false negatives (since ordering of the results is not important)
		sort.Strings(completions)
		sort.Strings(tt.output)

		if !reflect.DeepEqual(tt.output, completions) {
			t.Errorf("expected output: %#v,got: %#v", tt.output, completions)
		}
	}
}

func TestServiceParameterCompletionHandler(t *testing.T) {
	serviceClassList := &scv1beta1.ClusterServiceClassList{
		Items: []scv1beta1.ClusterServiceClass{testingutil.FakeClusterServiceClass("class name", "dummy")},
	}
	tests := []struct {
		name                 string
		returnedServiceClass *scv1beta1.ClusterServiceClassList
		returnedServicePlan  []scv1beta1.ClusterServicePlan
		output               []string
		parsedArgs           parsedArgs
	}{
		{
			name: "Case 0: no service name supplied",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create"},
				},
			},
			output: []string{},
		},
		{
			name:                 "Case 1: no plan supplied and single plan exists",
			returnedServiceClass: serviceClassList,
			returnedServicePlan:  []scv1beta1.ClusterServicePlan{testingutil.FakeClusterServicePlan("default", 1)},
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create", "class name"},
				},
			},
			output: []string{"PLAN_DATABASE_URI", "PLAN_DATABASE_USERNAME", "PLAN_DATABASE_PASSWORD", "SOME_OTHER"},
		},
		{
			name:                 "Case 2: no plan supplied and multiple plans exists",
			returnedServiceClass: serviceClassList,
			returnedServicePlan: []scv1beta1.ClusterServicePlan{
				testingutil.FakeClusterServicePlan("plan1", 1),
				testingutil.FakeClusterServicePlan("plan2", 2),
			},
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create", "class name"},
				},
			},
			output: []string{},
		},
		{
			name:                 "Case 3: plan supplied but doesn't match",
			returnedServiceClass: serviceClassList,
			returnedServicePlan:  []scv1beta1.ClusterServicePlan{testingutil.FakeClusterServicePlan("default", 1)},
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create", "class name"},
				},
				flagValues: map[string]string{"plan": "other"},
			},
			output: []string{},
		},
		{
			name:                 "Case 4: matching plan supplied and no other parameters supplied",
			returnedServiceClass: serviceClassList,
			returnedServicePlan: []scv1beta1.ClusterServicePlan{
				testingutil.FakeClusterServicePlan("plan2", 2),
				testingutil.FakeClusterServicePlan("plan1", 1),
			},
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create", "class name"},
				},
				flagValues: map[string]string{"plan": "plan1"},
			},
			output: []string{"PLAN_DATABASE_URI", "PLAN_DATABASE_USERNAME", "PLAN_DATABASE_PASSWORD", "SOME_OTHER"},
		},
		{
			name:                 "Case 5: no plan supplied but some other parameters supplied",
			returnedServiceClass: serviceClassList,
			returnedServicePlan:  []scv1beta1.ClusterServicePlan{testingutil.FakeClusterServicePlan("default", 1)},
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create", "class name"},
				},
				flagValues: map[string]string{"parameters": "[PLAN_DATABASE_USERNAME, SOME_OTHER]"},
			},
			output: []string{"PLAN_DATABASE_URI", "PLAN_DATABASE_PASSWORD"},
		},
		{
			name:                 "Case 6: matching plan supplied but some other parameters supplied",
			returnedServiceClass: serviceClassList,
			returnedServicePlan:  []scv1beta1.ClusterServicePlan{testingutil.FakeClusterServicePlan("default", 1)},
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create", "class name"},
				},
				flagValues: map[string]string{"plan": "default", "parameters": "[PLAN_DATABASE_USERNAME]"},
			},
			output: []string{"PLAN_DATABASE_URI", "PLAN_DATABASE_PASSWORD", "SOME_OTHER"},
		},
	}

	for _, tt := range tests {
		client, fakeClientSet := occlient.FakeNew()
		context := genericclioptions.NewFakeContext("project", "app", "component", client)

		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "clusterserviceclasses", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, tt.returnedServiceClass, nil
		})

		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "clusterserviceplans", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &scv1beta1.ClusterServicePlanList{Items: tt.returnedServicePlan}, nil
		})

		completions := ServiceParameterCompletionHandler(nil, tt.parsedArgs, context)

		// Sort the output and expected output in order to avoid false negatives (since ordering of the results is not important)
		sort.Strings(completions)
		sort.Strings(tt.output)

		if !reflect.DeepEqual(tt.output, completions) {
			t.Errorf("expected output: %#v,got: %#v", tt.output, completions)
		}
	}
}

func TestLinkCompletionHandler(t *testing.T) {

	tests := []struct {
		name        string
		component   string
		dcList      appsv1.DeploymentConfigList
		serviceList scv1beta1.ServiceInstanceList
		output      []string
	}{
		{
			name:      "Case 1: both components and services are present",
			component: "frontend",
			serviceList: scv1beta1.ServiceInstanceList{
				Items: []scv1beta1.ServiceInstance{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mysql-persistent",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "mysql-persistent",
								componentlabels.ComponentTypeLabel: "mysql-persistent",
							},
							Annotations: map[string]string{
								component.ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: scv1beta1.ServiceInstanceSpec{
							PlanReference: scv1beta1.PlanReference{
								ClusterServiceClassExternalName: "mysql-persistent",
								ClusterServicePlanExternalName:  "default",
							},
						},
						Status: scv1beta1.ServiceInstanceStatus{
							Conditions: []scv1beta1.ServiceInstanceCondition{
								{
									Reason: "ProvisionedSuccessfully",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "postgresql-ephemeral",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "postgresql-ephemeral",
								componentlabels.ComponentTypeLabel: "postgresql-ephemeral",
							},
							Annotations: map[string]string{
								component.ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: scv1beta1.ServiceInstanceSpec{
							PlanReference: scv1beta1.PlanReference{
								ClusterServiceClassExternalName: "postgresql-ephemeral",
								ClusterServicePlanExternalName:  "default",
							},
						},
						Status: scv1beta1.ServiceInstanceStatus{
							Conditions: []scv1beta1.ServiceInstanceCondition{
								{
									Reason: "Provisioning",
								},
							},
						},
					},
				},
			},
			dcList: appsv1.DeploymentConfigList{
				Items: []appsv1.DeploymentConfig{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "backend",
								componentlabels.ComponentTypeLabel: "java",
							},
							Annotations: map[string]string{
								component.ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "frontend",
								componentlabels.ComponentTypeLabel: "nodejs",
							},
							Annotations: map[string]string{
								component.ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
				},
			},
			// make sure that the 'component' is not part of the suggestions
			output: []string{"backend", "mysql-persistent", "postgresql-ephemeral"},
		},
	}

	for _, tt := range tests {
		client, fakeClientSet := occlient.FakeNew()
		parsedArgs := parsedArgs{
			commands: make(map[string]bool),
		}
		context := genericclioptions.NewFakeContext("project", "app", tt.component, client)

		//fake the services
		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "serviceinstances", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, &tt.serviceList, nil
		})

		//fake the dcs
		fakeClientSet.AppsClientset.PrependReactor("list", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, &tt.dcList, nil
		})

		for i := range tt.dcList.Items {
			fakeClientSet.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, &tt.dcList.Items[i], nil
			})
		}

		completions := LinkCompletionHandler(nil, parsedArgs, context)
		sort.Strings(completions)

		if !reflect.DeepEqual(tt.output, completions) {
			t.Errorf("expected output: %#v,got: %#v", tt.output, completions)
		}
	}
}

func TestUnlinkCompletionHandler(t *testing.T) {

	tests := []struct {
		name        string
		component   string
		dcList      appsv1.DeploymentConfigList
		serviceList scv1beta1.ServiceInstanceList
		output      []string
	}{
		{
			name:      "Case 1: both components and services are present",
			component: "frontend",
			serviceList: scv1beta1.ServiceInstanceList{
				Items: []scv1beta1.ServiceInstance{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mysql-persistent",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "mysql-persistent",
								componentlabels.ComponentTypeLabel: "mysql-persistent",
							},
						},
						Spec: scv1beta1.ServiceInstanceSpec{
							PlanReference: scv1beta1.PlanReference{
								ClusterServiceClassExternalName: "mysql-persistent",
								ClusterServicePlanExternalName:  "default",
							},
						},
						Status: scv1beta1.ServiceInstanceStatus{
							Conditions: []scv1beta1.ServiceInstanceCondition{
								{
									Reason: "ProvisionedSuccessfully",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "postgresql-ephemeral",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "postgresql-ephemeral",
								componentlabels.ComponentTypeLabel: "postgresql-ephemeral",
							},
						},
						Spec: scv1beta1.ServiceInstanceSpec{
							PlanReference: scv1beta1.PlanReference{
								ClusterServiceClassExternalName: "postgresql-ephemeral",
								ClusterServicePlanExternalName:  "default",
							},
						},
						Status: scv1beta1.ServiceInstanceStatus{
							Conditions: []scv1beta1.ServiceInstanceCondition{
								{
									Reason: "ProvisionedSuccessfully",
								},
							},
						},
					},
				},
			},
			dcList: appsv1.DeploymentConfigList{
				Items: []appsv1.DeploymentConfig{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backend-app",
							Namespace: "project",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "backend",
								componentlabels.ComponentTypeLabel: "java",
							},
							Annotations: map[string]string{
								component.ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backend2-app",
							Namespace: "project",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "backend2",
								componentlabels.ComponentTypeLabel: "java",
							},
							Annotations: map[string]string{
								component.ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
										},
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "frontend-app",
							Namespace: "project",
							Labels: map[string]string{
								applabels.ApplicationLabel:         "app",
								componentlabels.ComponentLabel:     "frontend",
								componentlabels.ComponentTypeLabel: "nodejs",
							},
							Annotations: map[string]string{
								component.ComponentSourceTypeAnnotation: "local",
							},
						},
						Spec: appsv1.DeploymentConfigSpec{
							Template: &corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "dummyContainer",
											EnvFrom: []corev1.EnvFromSource{
												{
													SecretRef: &corev1.SecretEnvSource{
														LocalObjectReference: corev1.LocalObjectReference{Name: "postgresql-ephemeral"},
													},
												},
												{
													SecretRef: &corev1.SecretEnvSource{
														LocalObjectReference: corev1.LocalObjectReference{Name: "backend-8080"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			// make sure that the 'component' is not part of the suggestions and that only actually linked components/services show up
			output: []string{"backend", "postgresql-ephemeral"},
		},
	}

	p := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "postgresql-ephemeral",
		},
	}

	for _, tt := range tests {
		client, fakeClientSet := occlient.FakeNew()
		parsedArgs := parsedArgs{
			commands: make(map[string]bool),
		}
		context := genericclioptions.NewFakeContext("project", "app", tt.component, client)

		//fake the services
		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "serviceinstances", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, &tt.serviceList, nil
		})

		//fake the dcs
		fakeClientSet.AppsClientset.PrependReactor("list", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, &tt.dcList, nil
		})

		for i := range tt.dcList.Items {
			fakeClientSet.AppsClientset.PrependReactor("get", "deploymentconfigs", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, &tt.dcList.Items[i], nil
			})
		}

		fakeClientSet.Kubernetes.PrependReactor("get", "secrets", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, &p, nil
		})

		completions := UnlinkCompletionHandler(nil, parsedArgs, context)
		sort.Strings(completions)

		if !reflect.DeepEqual(tt.output, completions) {
			t.Errorf("expected output: %#v,got: %#v", tt.output, completions)
		}
	}
}

func TestServiceCompletionHandler(t *testing.T) {
	tests := []struct {
		name                          string
		returnedServiceClassInstances *scv1beta1.ServiceInstanceList
		output                        []string
		parsedArgs                    parsedArgs
	}{
		{
			name: "test case 1: no service instance exists and name not typed",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"delete"},
				},
			},
			returnedServiceClassInstances: &scv1beta1.ServiceInstanceList{},
			output:                        []string{},
		},
		{
			name: "test case 2: one service class instance exists and name not typed",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"delete"},
				},
			},
			returnedServiceClassInstances: &scv1beta1.ServiceInstanceList{
				Items: []scv1beta1.ServiceInstance{
					testingutil.FakeServiceClassInstance("service-1", "mariadb-apb", "default", "ProvisionedSuccessfully"),
				},
			},
			output: []string{"service-1"},
		},
		{
			name: "test case 3: multiple service class instance exists and name not typed",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"delete"},
				},
			},
			returnedServiceClassInstances: &scv1beta1.ServiceInstanceList{
				Items: []scv1beta1.ServiceInstance{
					testingutil.FakeServiceClassInstance("service-1", "mariadb-apb", "default", "ProvisionedSuccessfully"),
					testingutil.FakeServiceClassInstance("service-2", "mariadb-apb", "prod", "ProvisionedSuccessfully"),
				},
			},
			output: []string{"service-1", "service-2"},
		},
		{
			name: "test case 4: multiple service class instance exists and name fully typed",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"delete"},
				},
				commands: map[string]bool{"service-1": true},
			},
			returnedServiceClassInstances: &scv1beta1.ServiceInstanceList{
				Items: []scv1beta1.ServiceInstance{
					testingutil.FakeServiceClassInstance("service-1", "mariadb-apb", "default", "ProvisionedSuccessfully"),
					testingutil.FakeServiceClassInstance("service-2", "mariadb-apb", "prod", "ProvisionedSuccessfully"),
				},
			},
			output: nil,
		},
	}

	for _, tt := range tests {
		client, fakeClientSet := occlient.FakeNew()
		context := genericclioptions.NewFakeContext("project", "app", "component", client)

		//fake the services
		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "serviceinstances", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, tt.returnedServiceClassInstances, nil
		})

		completions := ServiceCompletionHandler(nil, tt.parsedArgs, context)

		// Sort the output and expected output in order to avoid false negatives (since ordering of the results is not important)
		sort.Strings(completions)
		sort.Strings(tt.output)

		if !reflect.DeepEqual(tt.output, completions) {
			t.Errorf("expected output: %#v,got: %#v", tt.output, completions)
		}
	}
}

func TestServiceClassCompletionHandler(t *testing.T) {
	tests := []struct {
		name                   string
		returnedServiceClasses *scv1beta1.ClusterServiceClassList
		output                 []string
		parsedArgs             parsedArgs
	}{
		{
			name: "test case 1: no service class exists and name not typed",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create"},
				},
			},
			returnedServiceClasses: &scv1beta1.ClusterServiceClassList{},
			output:                 []string{},
		},
		{
			name: "test case 2: one service class exists and name not typed",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create"},
				},
			},
			returnedServiceClasses: &scv1beta1.ClusterServiceClassList{
				Items: []scv1beta1.ClusterServiceClass{
					testingutil.FakeClusterServiceClass("foo"),
				},
			},
			output: []string{"foo"},
		},
		{
			name: "test case 3: multiple service classes exist and name not typed",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"create"},
				},
			},
			returnedServiceClasses: &scv1beta1.ClusterServiceClassList{
				Items: []scv1beta1.ClusterServiceClass{
					testingutil.FakeClusterServiceClass("foo"),
					testingutil.FakeClusterServiceClass("bar"),
				},
			},
			output: []string{"foo", "bar"},
		},
		{
			name: "test case 4: multiple service classes exist and name fully typed",
			parsedArgs: parsedArgs{
				original: complete.Args{
					Completed: []string{"delete"},
				},
				commands: map[string]bool{"foo": true},
			},
			returnedServiceClasses: &scv1beta1.ClusterServiceClassList{
				Items: []scv1beta1.ClusterServiceClass{
					testingutil.FakeClusterServiceClass("foo"),
					testingutil.FakeClusterServiceClass("bar"),
				},
			},
			output: nil,
		},
	}

	for _, tt := range tests {
		client, fakeClientSet := occlient.FakeNew()
		context := genericclioptions.NewFakeContext("project", "app", "component", client)

		//fake the services
		fakeClientSet.ServiceCatalogClientSet.PrependReactor("list", "clusterserviceclasses", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, tt.returnedServiceClasses, nil
		})

		completions := ServiceClassCompletionHandler(nil, tt.parsedArgs, context)

		// Sort the output and expected output in order to avoid false negatives (since ordering of the results is not important)
		sort.Strings(completions)
		sort.Strings(tt.output)

		if !reflect.DeepEqual(tt.output, completions) {
			t.Errorf("expected output: %#v,got: %#v", tt.output, completions)
		}
	}
}
