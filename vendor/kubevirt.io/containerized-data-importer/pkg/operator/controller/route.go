/*
Copyright 2018 The CDI Authors.

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

package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	routev1 "github.com/openshift/api/route/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	uploadProxyServiceName = "cdi-uploadproxy"
	uploadProxyRouteName   = uploadProxyServiceName
	uploadProxyCASecret    = "cdi-upload-proxy-ca-key"
)

func ensureUploadProxyRouteExists(logger logr.Logger, c client.Client, scheme *runtime.Scheme, owner metav1.Object) error {
	namespace := owner.GetNamespace()
	if namespace == "" {
		return fmt.Errorf("cluster scoped owner not supported")
	}

	key := client.ObjectKey{Namespace: namespace, Name: uploadProxyRouteName}

	err := c.Get(context.TODO(), key, &routev1.Route{})
	if err == nil {
		// route already exists, so do nothing (user can mutate this)
		return nil
	}

	if meta.IsNoMatchError(err) {
		// not in openshift
		logger.V(3).Info("No match error for Route, must not be in openshift")
		return nil
	}

	if !errors.IsNotFound(err) {
		return err
	}

	secret := &corev1.Secret{}
	key = client.ObjectKey{Namespace: namespace, Name: uploadProxyCASecret}

	if err = c.Get(context.TODO(), key, secret); err != nil {
		return err
	}

	cert, exists := secret.Data["tls.crt"]
	if !exists {
		return fmt.Errorf("unexpected secret format, 'tls.crt' key missing")
	}

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uploadProxyRouteName,
			Namespace: namespace,
			Annotations: map[string]string{
				// long timeout here to make sure client conection doesn't die during qcow->raw conversion
				"haproxy.router.openshift.io/timeout": "60m",
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: uploadProxyServiceName,
			},
			TLS: &routev1.TLSConfig{
				Termination:              routev1.TLSTerminationReencrypt,
				DestinationCACertificate: string(cert),
			},
		},
	}

	if err = controllerutil.SetControllerReference(owner, route, scheme); err != nil {
		return err
	}

	if err = c.Create(context.TODO(), route); err != nil {
		return err
	}

	return nil
}
