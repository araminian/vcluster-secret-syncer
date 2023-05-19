package syncers

import (
	"fmt"

	"github.com/araminian/vcluster-secret-syncer/constants"
	"github.com/loft-sh/vcluster-sdk/translate"

	"github.com/loft-sh/vcluster-sdk/syncer"
	synccontext "github.com/loft-sh/vcluster-sdk/syncer/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ManagedSecret = "plugin.vcluster.loft.sh/managed-by"
)

type secretSyncer struct {
	hostNamespace string
}

func NewSecretSyncer(ctx *synccontext.RegisterContext) syncer.Syncer {

	return &secretSyncer{
		hostNamespace: ctx.TargetNamespace,
	}

}

func (s *secretSyncer) Name() string {
	return constants.PluginName
}

func (s *secretSyncer) Resource() client.Object {
	return &corev1.Secret{}
}

// make sure to impelement syncer.starter interface
var _ syncer.Starter = &secretSyncer{}

// reconcile start
func (s *secretSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	return true, nil
}

func (s *secretSyncer) ReconcileEnd() {
	// NOOP
}

// make sure to implement syncer.Syncer interface
var _ syncer.UpSyncer = &secretSyncer{}

func (s *secretSyncer) SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {

	pSecret := pObj.(*corev1.Secret)

	// ignore secrets that are sync from vcluster to host
	if pSecret.GetLabels()[translate.MarkerLabel] != "" {
		return ctrl.Result{}, nil
	}

	// check if the secret has the right annotation
	secretAnnotations := pSecret.GetAnnotations()

	// check if the EnableSyncAnnotation is set and is true
	if secretAnnotations[constants.EnableSyncAnnotation] != "true" {
		return ctrl.Result{}, nil
	}

	// check if the DestinationNamespaceAnnotation is set
	secretDestinationNamespace := secretAnnotations[constants.DestinationNamespaceAnnotation]
	if secretDestinationNamespace == "" {
		err := fmt.Errorf("failed to create secret %s/%s: '%s' annotation not set", pSecret.GetNamespace(), pSecret.GetName(), constants.DestinationNamespaceAnnotation)
		return ctrl.Result{}, err
	}

	labels := map[string]string{
		ManagedSecret: constants.PluginName,
	}

	for k, v := range pSecret.GetLabels() {
		labels[k] = v
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pSecret.GetName(),
			Namespace:   secretDestinationNamespace,
			Labels:      labels,
			Annotations: pSecret.GetAnnotations(),
		},
		Immutable: pSecret.Immutable,
		Data:      pSecret.Data,
		Type:      pSecret.Type,
	}

	err := ctx.VirtualClient.Create(ctx.Context, secret)
	if err == nil {
		ctx.Log.Infof("created secret %s/%s", secretDestinationNamespace, secret.GetName())

	} else {
		ctx.Log.Errorf("failed to create secret %s/%s: %v", secretDestinationNamespace, secret.GetName(), err)
	}

	return ctrl.Result{}, err

}

func (s *secretSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {

	pSecret := pObj.(*corev1.Secret)

	// check if the secret has the right annotation
	secretAnnotations := pSecret.GetAnnotations()

	// check if the EnableSyncAnnotation is set and is true
	if secretAnnotations[constants.EnableSyncAnnotation] != "true" {

		// check if the secret is managed by this plugin
		if vObj.GetLabels()[ManagedSecret] == constants.PluginName {
			err := ctx.VirtualClient.Delete(ctx.Context, vObj)

			if err == nil {
				ctx.Log.Infof("deleted secret %s/%s", vObj.GetNamespace(), vObj.GetName())
			} else {
				ctx.Log.Errorf("failed to delete secret %s/%s: %v", vObj.GetNamespace(), vObj.GetName(), err)
			}

			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	vSecret := vObj.(*corev1.Secret)

	updated := s.translateUpdateUp(pSecret, vSecret)
	if updated == nil {
		// no update needed
		return ctrl.Result{}, nil
	}

	err := ctx.VirtualClient.Update(ctx.Context, updated)

	if err == nil {
		ctx.Log.Infof("updated secret %s/%s", updated.GetNamespace(), updated.GetName())
	} else {
		ctx.Log.Errorf("failed to update secret %s/%s: %v", updated.GetNamespace(), updated.GetName(), err)
	}
	return ctrl.Result{}, err

}

func (s *secretSyncer) translateUpdateUp(pObj, vObj *corev1.Secret) *corev1.Secret {

	var updated *corev1.Secret

	// sync annotations
	if !equality.Semantic.DeepEqual(pObj.GetAnnotations(), vObj.GetAnnotations()) {
		updated = newIfNil(updated, vObj)
		updated.Annotations = pObj.GetAnnotations()
	}

	// check labels
	expectedLabels := map[string]string{
		ManagedSecret: constants.PluginName,
	}
	for k, v := range pObj.GetLabels() {
		expectedLabels[k] = v
	}

	if !equality.Semantic.DeepEqual(expectedLabels, vObj.GetLabels()) {
		updated = newIfNil(updated, vObj)
		updated.Labels = expectedLabels
	}

	// check data
	if !equality.Semantic.DeepEqual(vObj.Data, pObj.Data) {
		updated = newIfNil(updated, vObj)
		updated.Data = pObj.Data
	}

	return updated

}

func newIfNil(updated *corev1.Secret, pObj *corev1.Secret) *corev1.Secret {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}

func (s *secretSyncer) IsManaged(pObj client.Object) (bool, error) {
	// we will consider all Secrets as managed in order to reconcile
	// when a secret type changes, and we will check the type
	// in the Sync and SyncUp methods and ignore the irrelevant ones
	return true, nil
}

// VirtualToPhysical translates a virtual name to a physical name
func (s *secretSyncer) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	// the secret that is being mirrored by a particular vObj secret
	// is located in the "HostNamespace" of the host cluster
	return types.NamespacedName{
		Namespace: s.hostNamespace,
		Name:      req.Name,
	}
}

// PhysicalToVirtual translates a physical name to a virtual name
func (s *secretSyncer) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	// the secret mirrored to vcluster is always named the same as the
	// original in the host, and it is located in the DestinationNamespace
	return types.NamespacedName{
		Namespace: s.DestinationNamespace(pObj),
		Name:      pObj.GetName(),
	}
}

func (s *secretSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {

	// this is called when the secret in the host gets removed
	// or if the vObj is an unrelated Secret created in vcluster

	// check if this particular secret was created by this plugin
	if vObj.GetLabels()[ManagedSecret] == constants.PluginName {
		// delete the secret in the vcluster
		err := ctx.VirtualClient.Delete(ctx.Context, vObj)

		if err == nil {
			ctx.Log.Infof("deleted secret %s/%s", vObj.GetNamespace(), vObj.GetName())
		} else {
			ctx.Log.Errorf("failed to delete secret %s/%s: %v", vObj.GetNamespace(), vObj.GetName(), err)
		}
		return ctrl.Result{}, err

	}

	// ignore all unrelated Secrets
	return ctrl.Result{}, nil
}

// Find destinaton namespace
func (s *secretSyncer) DestinationNamespace(pObj client.Object) string {

	pSecret := pObj.(*corev1.Secret)

	// check if the secret has the right annotation
	secretAnnotations := pSecret.GetAnnotations()

	// check if the EnableSyncAnnotation is set and is true
	if secretAnnotations[constants.EnableSyncAnnotation] != "true" {
		return ""
	}

	// check if the DestinationNamespaceAnnotation is set
	secretDestinationNamespace := secretAnnotations[constants.DestinationNamespaceAnnotation]
	if secretDestinationNamespace == "" {
		return ""
	}

	return secretDestinationNamespace
}
