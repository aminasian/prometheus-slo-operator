/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	monitoringv1alpha1 "github.com/aminasian/prometheus-slo-operator/api/v1alpha1"
	promoperatorv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

// ServiceLevelReconciler reconciles a ServiceLevel object
type ServiceLevelReconciler struct {
	client.Client
	Log                    logr.Logger
	Scheme                 *runtime.Scheme
	SLOCalculatorContainer string
	IsPrometheusOperator   bool
	UseVPAResource         bool
}

// +kubebuilder:rbac:groups=monitoring.aminasian.com,resources=servicelevels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.aminasian.com,resources=servicelevels/status,verbs=get;update;patch

func (r *ServiceLevelReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("servicelevel", req.NamespacedName)

	// your logic here

	// Fetch the ServiceLevel instance
	instance := &monitoringv1alpha1.ServiceLevel{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	deployment, err := newDeploymentResourceForCR(instance, r.SLOCalculatorContainer, r.IsPrometheusOperator)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := controllerutil.SetControllerReference(instance, deployment, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	pdb, err := newPDBResourceForCR(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := controllerutil.SetControllerReference(instance, pdb, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	cm, err := newConfigMapResourceForCR(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := controllerutil.SetControllerReference(instance, cm, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// ConfigMap needs to be created before the Deployment resource to prevent a ContainerConfigErr race condition
	foundCM := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, foundCM)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		// handle update
	}

	// Deployment
	foundDeployment := &v1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.Create(ctx, deployment)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		// handle update
	}

	// PDB
	foundPDB := &v1beta1.PodDisruptionBudget{}
	err = r.Get(ctx, types.NamespacedName{Name: pdb.Name, Namespace: pdb.Namespace}, foundPDB)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new PDB", "PDB.Namespace", pdb.Namespace, "PDB.Name", pdb.Name)
		err = r.Create(ctx, pdb)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		// handle update
	}

	//service, err := newServiceResourceForCR(instance)
	//if err != nil {
	//	return reconcile.Result{}, err
	//}

	//

	//jsonBytes, _ := json.Marshal(prometheusRule)
	//go_log.Printf("PrometheusRule: \n\n\n\n %v \n\n\n\n", string(jsonBytes))

	if r.IsPrometheusOperator {

		prometheusRule, err := newPrometheusRuleResourceForCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		if err := controllerutil.SetControllerReference(instance, prometheusRule, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		//podMonitor, err := newPodMonitorForCR(instance)
		//if err != nil {
		//	return reconcile.Result{}, err
		//}
		//
		//if err := controllerutil.SetControllerReference(instance, podMonitor, r.Scheme); err != nil {
		//	return reconcile.Result{}, err
		//}

		service, err := newServiceResourceForCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		if err := controllerutil.SetControllerReference(instance, service, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		serviceMonitor, err :=  newServiceMonitorResourceForCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		if err := controllerutil.SetControllerReference(instance, serviceMonitor, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		// PrometheusRule
		foundPrometheusRule := &promoperatorv1.PrometheusRule{}
		err = r.Get(ctx, types.NamespacedName{Name: prometheusRule.Name, Namespace: prometheusRule.Namespace}, foundPrometheusRule)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new PrometheusRule", "PrometheusRule.Namespace", prometheusRule.Namespace, "PrometheusRule.Name", prometheusRule.Name)
			err = r.Create(ctx, prometheusRule)
			if err != nil {
				return reconcile.Result{}, err
			}

		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// handle update
		}

		//// PodMonitor
		//foundPodMonitor := &promoperatorv1.PodMonitor{}
		//err = r.Get(ctx, types.NamespacedName{Name: podMonitor.Name, Namespace: podMonitor.Namespace}, foundPodMonitor)
		//if err != nil && errors.IsNotFound(err) {
		//	logger.Info("Creating a new PrometheusRule", "PodMonitor.Namespace", podMonitor.Namespace, "PodMonitor.Name", podMonitor.Name)
		//	err = r.Create(ctx, podMonitor)
		//	if err != nil {
		//		return reconcile.Result{}, err
		//	}
		//} else if err != nil {
		//	return reconcile.Result{}, err
		//} else {
		//	// handle update
		//}

		// Service
		foundService := &corev1.Service{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			err = r.Create(context.TODO(), service)
			if err != nil {
				return reconcile.Result{}, err
			}

		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// handle update
		}

		// ServiceMonitor
		foundServiceMonitor := &promoperatorv1.ServiceMonitor{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: serviceMonitor.Name, Namespace: serviceMonitor.Namespace}, foundServiceMonitor)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new ServiceMonitor", "ServiceMonitor.Namespace", serviceMonitor.Namespace, "ServiceMonitor.Name", serviceMonitor.Name)
			err = r.Create(context.TODO(), serviceMonitor)
			if err != nil {
				return reconcile.Result{}, err
			}

		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// handle update
		}

	}

	if r.UseVPAResource {

		vpa, err := newVPAResourceForCR(deployment)
		if err != nil {
			return reconcile.Result{}, err
		}

		if err := controllerutil.SetControllerReference(instance, vpa, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		// VPA
		foundVPA := &vpav1.VerticalPodAutoscaler{}
		err = r.Get(ctx, types.NamespacedName{Name: vpa.Name, Namespace: vpa.Namespace}, foundVPA)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new PDB", "PDB.Namespace", vpa.Namespace, "PDB.Name", vpa.Name)
			err = r.Create(ctx, vpa)
			if err != nil {
				return reconcile.Result{}, err
			}

		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// handle update
		}

	}

	return ctrl.Result{}, nil
}

func (r *ServiceLevelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.ServiceLevel{}).
		Complete(r)
}

func newDeploymentResourceForCR(cr *monitoringv1alpha1.ServiceLevel, calculatorContainer string, isPromOperator bool) (*v1.Deployment, error) {
	cpuLimit, err := resource.ParseQuantity("50m")
	if err != nil {
		return nil, err
	}
	cpuRequested, err := resource.ParseQuantity("25m")
	if err != nil {
		return nil, err
	}
	memLimit, err := resource.ParseQuantity("52Mi")
	if err != nil {
		return nil, err
	}
	memReq, err := resource.ParseQuantity("7Mi")
	if err != nil {
		return nil, err
	}

	maxUnavailable := intstr.Parse("1")
	var replicas int32 = 1
	var port int32 = 8080

	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    cr.ObjectMeta.Labels,
		},
		Spec: v1.DeploymentSpec{

			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": cr.ObjectMeta.Labels["app.kubernetes.io/name"]},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   cr.Name,
					Labels: cr.ObjectMeta.Labels,
				},
				Spec: corev1.PodSpec{
					//DNSConfig: &corev1.PodDNSConfig{
					//	Options: []corev1.PodDNSConfigOption{
					//		{
					//			Name:  "ndots",
					//			Value: &nDotsValue,
					//		},
					//		{
					//			Name: "single-request-reopen",
					//		},
					//	},
					//},
					Containers: []corev1.Container{{
						Name:  "iheart-slo-exporter",
						Image: calculatorContainer,
						Args:  []string{},
						Ports: []corev1.ContainerPort{{
							Name: "http",
							ContainerPort: port,
						}},
						//Env: {},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    cpuLimit,
								corev1.ResourceMemory: memLimit,
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    cpuRequested,
								corev1.ResourceMemory: memReq,
							},
						},
						ImagePullPolicy: "Always",
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "config",
								ReadOnly:  true,
								MountPath: "/app/config",
							},
						},
						//LivenessProbe:   &corev1.Probe{
						//	Handler: corev1.Handler{
						//		Exec: &corev1.ExecAction{
						//			Command: nil,
						//		},
						//		HTTPGet: &corev1.HTTPGetAction{
						//			Path: "/healthz/live",
						//			Port: intstr.IntOrString{
						//				Type:   0,
						//				IntVal: 0,
						//				StrVal: "8080",
						//			},
						//			Scheme:      "http",
						//		},
						//	},
						//	InitialDelaySeconds: 0,
						//	TimeoutSeconds:      0,
						//	PeriodSeconds:       0,
						//	SuccessThreshold:    0,
						//	FailureThreshold:    0,
						//},
						//ReadinessProbe:  &corev1.Probe{
						//	Handler: corev1.Handler{
						//		Exec: &corev1.ExecAction{
						//			Command: nil,
						//		},
						//		HTTPGet: &corev1.HTTPGetAction{
						//			Path: "/healthz/ready",
						//			Port: intstr.IntOrString{
						//				Type:   0,
						//				IntVal: 0,
						//				StrVal: "8080",
						//			},
						//			Scheme:      "http",
						//		},
						//	},
						//	InitialDelaySeconds: 0,
						//	TimeoutSeconds:      0,
						//	PeriodSeconds:       0,
						//	SuccessThreshold:    0,
						//	FailureThreshold:    0,
						//},
					}},
					RestartPolicy: "Always",
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{{
									MatchExpressions: []corev1.NodeSelectorRequirement{{
										Key:      "group",
										Operator: "In",
										Values:   []string{"gap"},
									}},
								}},
							},
						},
					},
					Tolerations: []corev1.Toleration{{
						Key:      "dedicated",
						Operator: "Equal",
						Value:    "gap",
						Effect:   "NoSchedule",
					}},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cr.Name,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "config.json",
											Path: "config.json",
										},
									},
									//DefaultMode: "",
									//Optional:    "",
								},
							},
						},
					},
				},
			},
			Strategy: v1.DeploymentStrategy{
				Type: "RollingUpdate",
				RollingUpdate: &v1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
				},
			},
		},
	}, nil
}

func newPrometheusRuleResourceForCR(cr *monitoringv1alpha1.ServiceLevel) (*promoperatorv1.PrometheusRule, error) {

	// generate prometheus rules for querying efficiency
	var rules []promoperatorv1.RuleGroup
	ruleGroup := promoperatorv1.RuleGroup{
		Name:                    fmt.Sprintf("%v-%v", cr.Name, "rules"),
		Interval:                "",
		Rules:                   []promoperatorv1.Rule{},
		PartialResponseStrategy: "",
	}
	// Generate Prometheus Rules for efficient SLI calculations
	for _, slo := range cr.Spec.ServiceLevelObjectives {
		splitSLOName := strings.Split(slo.Name, "-")
		promFormattedSLOName := ""
		for i, str := range splitSLOName {
			if i == 0 {
				promFormattedSLOName = str
			} else {
				promFormattedSLOName = fmt.Sprintf("%v_%v", promFormattedSLOName, str)
			}
		}
		totalQueryRuleName := fmt.Sprintf("%v:%v_%v:%v", "sli", promFormattedSLOName, "totalQuery", "increase5m:sum")
		totalRule := promoperatorv1.Rule{
			Record: totalQueryRuleName,
			Expr: intstr.IntOrString{
				Type:   1,
				IntVal: 0,
				StrVal: slo.ServiceLevelIndicator.Prometheus.TotalQuery,
			},
		}
		slo.ServiceLevelIndicator.Prometheus.TotalQuery = totalQueryRuleName

		errorQueryRuleName := fmt.Sprintf("%v:%v_%v:%v", "sli", promFormattedSLOName, "errorQuery", "increase5m:sum")
		errorRule := promoperatorv1.Rule{
			Record: errorQueryRuleName,
			Expr: intstr.IntOrString{
				Type:   1,
				IntVal: 0,
				StrVal: slo.ServiceLevelIndicator.Prometheus.ErrorQuery,
			},
		}
		slo.ServiceLevelIndicator.Prometheus.ErrorQuery = errorQueryRuleName

		ruleGroup.Rules = append(ruleGroup.Rules, totalRule)
		ruleGroup.Rules = append(ruleGroup.Rules, errorRule)
	}

	// Generate Prometheus Rules for error budget violation alert's PromQL expressions
	// and also add Alerts to Prometheus Rule resource

	// ------------------------------------------------------------------ //
	splitCRName := strings.Split(cr.Name, "-")
	promFormattedCRName := ""
	for i, str := range splitCRName {
		if i == 0 {
			promFormattedCRName = str
		} else {
			promFormattedCRName = fmt.Sprintf("%v_%v", promFormattedCRName, str)
		}
	}

	oneHourAlertExpr := fmt.Sprintf(
		"("+
			"increase(service_level_sli_result_error_ratio_total{service_level=\"%v\"}[1h])"+
			"/"+
			"increase(service_level_sli_result_count_total{service_level=\"%v\"}[1h])"+
			") > (1 - service_level_slo_objective_ratio{service_level=\"%v\"}) * 0.02 "+
			"and"+
			" ("+
			"increase(service_level_sli_result_error_ratio_total{service_level=\"%v\"}[5m])"+
			"/"+
			"increase(service_level_sli_result_count_total{service_level=\"%v\"}[5m])"+
			") > (1 - service_level_slo_objective_ratio{service_level=\"%v\"}) * 0.02", cr.Name, cr.Name, cr.Name, cr.Name, cr.Name, cr.Name)

	oneHourAlertRuleName := fmt.Sprintf("alert:%v:%v", promFormattedCRName, "SLOErrorRateTooFast1h")

	oneHourAlertRule := promoperatorv1.Rule{
		Record: oneHourAlertRuleName,
		Expr: intstr.IntOrString{
			Type:   1,
			IntVal: 0,
			StrVal: oneHourAlertExpr,
		},
	}

	ruleGroup.Rules = append(ruleGroup.Rules, oneHourAlertRule)

	// Generate Alerts
	oneHourAlert := promoperatorv1.Rule{
		Alert: fmt.Sprintf("%v-%v", cr.Name, "SLOErrorRateTooFast1h"),
		Expr: intstr.IntOrString{
			Type:   1,
			IntVal: 0,
			StrVal: oneHourAlertRuleName,
		},
		For:    "15m",
		Labels: map[string]string{"severity": "critical", "owner": ""},
		Annotations: map[string]string{
			"summary":     "The SLO error budget burn rate for 1h is greater than 2%",
			"description": "The error rate for 1h in the {{$labels.service_level}}/{{$labels.slo}} SLO error budget is too fast, is greater than the total error budget 2%.",
		},
	}

	sixHourAlertExpr := fmt.Sprintf(
		"("+
			"increase(service_level_sli_result_error_ratio_total{service_level=\"%v\"}[6h])"+
			"/"+
			"increase(service_level_sli_result_count_total{service_level=\"%v\"}[6h])"+
			") > (1 - service_level_slo_objective_ratio{service_level=\"%v\"}) * 0.05 "+
			"and"+
			" ("+
			"increase(service_level_sli_result_error_ratio_total{service_level=\"%v\"}[30m])"+
			"/"+
			"increase(service_level_sli_result_count_total{service_level=\"%v\"}[30m])"+
			") > (1 - service_level_slo_objective_ratio{service_level=\"%v\"}) * 0.05", cr.Name, cr.Name, cr.Name, cr.Name, cr.Name, cr.Name)

	sixHourAlertRuleName := fmt.Sprintf("alert:%v:%v", promFormattedCRName, "SLOErrorRateTooFast6h")

	sixHourAlertRule := promoperatorv1.Rule{
		Record: oneHourAlertRuleName,
		Expr: intstr.IntOrString{
			Type:   1,
			IntVal: 0,
			StrVal: sixHourAlertExpr,
		},
	}

	ruleGroup.Rules = append(ruleGroup.Rules, sixHourAlertRule)

	sixHourAlert := promoperatorv1.Rule{
		Alert: fmt.Sprintf("%v-%v", cr.Name, "SLOErrorRateTooFast6h"),
		Expr: intstr.IntOrString{
			Type:   1,
			IntVal: 0,
			StrVal: sixHourAlertRuleName,
		},
		For:    "15m",
		Labels: map[string]string{"severity": "critical", "owner": ""},
		Annotations: map[string]string{
			"summary":     "The SLO error budget burn rate for 6h is greater than 5%",
			"description": "The error rate for 6h in the {{$labels.service_level}}/{{$labels.slo}} SLO error budget is too fast, is greater than the total error budget 5%.",
		},
	}

	ruleGroup.Rules = append(ruleGroup.Rules, oneHourAlert)
	ruleGroup.Rules = append(ruleGroup.Rules, sixHourAlert)

	rules = append(rules, ruleGroup)

	labels := cr.ObjectMeta.Labels
	labels["prometheus"] = cr.Spec.PrometheusName

	return &promoperatorv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PrometheusRule",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: promoperatorv1.PrometheusRuleSpec{
			Groups: rules,
		},
	}, nil
}

func newPDBResourceForCR(cr *monitoringv1alpha1.ServiceLevel) (*v1beta1.PodDisruptionBudget, error) {
	maxUnavailable := intstr.Parse("1")
	return &v1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		Spec: v1beta1.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxUnavailable,
		},
	}, nil
}

func newConfigMapResourceForCR(cr *monitoringv1alpha1.ServiceLevel) (*corev1.ConfigMap, error) {

	jsonBytes, err := json.Marshal(cr)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    cr.ObjectMeta.Labels,
		},
		Data: map[string]string{"config.json": string(jsonBytes)},
	}, nil
}

func newVPAResourceForCR(d *v1.Deployment) (*vpav1.VerticalPodAutoscaler, error) {

	cpuLimit, err := resource.ParseQuantity("5000m")
	if err != nil {
		return nil, err
	}

	memLimit, err := resource.ParseQuantity("10Gi")
	if err != nil {
		return nil, err
	}

	updateMode := vpav1.UpdateModeAuto

	return &vpav1.VerticalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VerticalPodAutoscaler",
			APIVersion: "autoscaling.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.Name,
			Namespace: d.Namespace,
			Labels:    d.ObjectMeta.Labels,
		},
		Spec: vpav1.VerticalPodAutoscalerSpec{
			TargetRef: &autoscalingv1.CrossVersionObjectReference{
				Kind:       d.Kind,
				Name:       d.Name,
				APIVersion: d.APIVersion,
			},
			UpdatePolicy: &vpav1.PodUpdatePolicy{
				UpdateMode: &updateMode,
			},
			ResourcePolicy: &vpav1.PodResourcePolicy{
				ContainerPolicies: []vpav1.ContainerResourcePolicy{{
					ContainerName: d.Spec.Template.Spec.Containers[0].Name,
					//Mode:          nil,
					//MinAllowed:    nil,
					MaxAllowed: corev1.ResourceList{
						corev1.ResourceCPU:    cpuLimit,
						corev1.ResourceMemory: memLimit,
					},
				}},
			},
		},
	}, nil
}

func newServiceResourceForCR(cr *monitoringv1alpha1.ServiceLevel) (*corev1.Service, error) {

	var port int32 = 8080

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      cr.ObjectMeta.Labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name: "http",
				Port: port,
			}},
			Selector: map[string]string{"app.kubernetes.io/name": cr.ObjectMeta.Labels["app.kubernetes.io/name"]},
			Type:     "ClusterIP",
		},
	}, nil
}

func newServiceMonitorResourceForCR(cr *monitoringv1alpha1.ServiceLevel) (*promoperatorv1.ServiceMonitor, error) {

	labels := cr.ObjectMeta.Labels
	labels["prometheus"] = cr.Spec.PrometheusName

	return &promoperatorv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceMonitor",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            cr.Name,
			Namespace:       cr.Namespace,
			Labels:          labels,

		},
		Spec: promoperatorv1.ServiceMonitorSpec{
			JobLabel:        "jobLabel",
			Endpoints:       []promoperatorv1.Endpoint{{
				Port: 		 "http",
				Path:        "/metrics",
			}},
			Selector: metav1.LabelSelector{
				MatchLabels:      map[string]string{"app.kubernetes.io/name": cr.ObjectMeta.Labels["app.kubernetes.io/name"]},
			},
			NamespaceSelector: promoperatorv1.NamespaceSelector{
				Any:        false,
				MatchNames: []string{cr.Namespace},
			},
		},
	}, nil
}

func newPodMonitorForCR(cr *monitoringv1alpha1.ServiceLevel) (*promoperatorv1.PodMonitor, error) {

	labels := cr.ObjectMeta.Labels
	labels["prometheus"] = cr.Spec.PrometheusName

	return &promoperatorv1.PodMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodMonitor",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: promoperatorv1.PodMonitorSpec{
			PodMetricsEndpoints: []promoperatorv1.PodMetricsEndpoint{{
				Port: "http",
				Path: "/metrics",
			}},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": cr.ObjectMeta.Labels["app.kubernetes.io/name"]},
			},
			NamespaceSelector: promoperatorv1.NamespaceSelector{
				Any:        false,
				MatchNames: []string{cr.Namespace},
			},
		},
	}, nil
}
