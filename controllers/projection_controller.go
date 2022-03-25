/*
Copyright 2022.

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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ProjectionReconciler reconciles a Demo object
type ProjectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=therealhaoliu.io.therealhaoliu.io,resources=demoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=therealhaoliu.io.therealhaoliu.io,resources=demoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=therealhaoliu.io.therealhaoliu.io,resources=demoes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Demo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ProjectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(req.Name)

	// get managedcluster
	managedCluster := clusterv1.ManagedCluster{}
	r.Client.Get(ctx, req.NamespacedName, &managedCluster)

	// determine if managedcluster has been imported
	managedClusterJoined := false
	for _, condition := range managedCluster.Status.Conditions {
		if condition.Type == clusterv1.ManagedClusterConditionJoined && condition.Status == "True" {
			managedClusterJoined = true
			break
		}
	}

	configmapNsN := types.NamespacedName{
		Namespace: "import-secret-projection",
		Name:      req.Name,
	}

	if managedClusterJoined {
		// if imported deleted the projected configmap
		configmap := corev1.ConfigMap{}
		err := r.Client.Get(ctx, configmapNsN, &configmap)
		if err != nil {
			r.Client.Delete(ctx, &configmap)
		}

	} else {
		// if not imported check if the projected configmap exist
		// get import secret
		importSecretNsN := types.NamespacedName{
			Namespace: req.Name,
			Name:      fmt.Sprintf("%s-import", req.Name),
		}
		importSecret := corev1.Secret{}
		err := r.Client.Get(ctx, importSecretNsN, &importSecret)
		if err != nil {
			return ctrl.Result{}, err
		}

		newProjectConfigmap := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      configmapNsN.Name,
				Namespace: configmapNsN.Namespace,
			},
			Data: make(map[string]string),
		}

		for key, value := range importSecret.Data {
			newProjectConfigmap.Data[key] = string(value)
		}

		projectConfigmap := corev1.ConfigMap{}
		err = r.Client.Get(ctx, configmapNsN, &projectConfigmap)
		if err != nil {
			if errors.IsNotFound(err) {
				err = r.Client.Create(ctx, &newProjectConfigmap)
			}
		}
		// TODO update if does not deep equal
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.ManagedCluster{}).
		Complete(r)
}
