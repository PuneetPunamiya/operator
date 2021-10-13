/*
Copyright 2021 The Tekton Authors

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

package v1alpha1

import (
	"context"
	"net/url"

	"knative.dev/pkg/apis"
)

func (tp *TektonHub) Validate(ctx context.Context) (errs *apis.FieldError) {

	if apis.IsInDelete(ctx) {
		return nil
	}

	if tp.Spec.TargetNamespace == "" {
		errs = errs.Also(apis.ErrMissingField("spec.targetNamespace"))
	}

	return errs.Also(tp.Spec.ApiSpec.validate("spec.api"), tp.Spec.DbSpec.validate("spec.db"))
}

func (api *ApiSpec) validate(path string) (errs *apis.FieldError) {

	if api.HubConfigUrl != "" {
		_, err := url.ParseRequestURI(api.HubConfigUrl)
		if err != nil {
			errs = errs.Also(apis.ErrInvalidValue(api.HubConfigUrl, path+".HubConfigUrl"))
		}
	}

	if api.HubConfigUrl == "" {
		errs = errs.Also(apis.ErrInvalidValue(api.HubConfigUrl, path+".HubConfigUrl"))
	}

	if api.ApiSecretName == "" {
		errs = errs.Also(apis.ErrInvalidValue(api.ApiSecretName, path+".ApiSecretName"))
	}

	return errs
}

func (db *DbSpec) validate(path string) (errs *apis.FieldError) {

	if db.DbSecretName == "" {
		errs = errs.Also(apis.ErrInvalidValue(db.DbSecretName, path+".DbSecretName"))
	}

	return errs
}
