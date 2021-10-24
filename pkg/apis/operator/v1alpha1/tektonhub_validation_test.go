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
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_ValidateTektonHub_MissingHubConfigUrl(t *testing.T) {

	th := &TektonHub{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: TektonHubSpec{
			Db: DbSpec{
				DbSecretName: "db",
			},
			Api: ApiSpec{
				ApiSecretName: "api",
			},
		},
	}

	err := th.Validate(context.TODO())
	assert.Equal(t, "missing field(s): spec.api.HubConfigUrl", err.Error())
}

func Test_ValidateTektonHub_MissingApiSecretName(t *testing.T) {

	th := &TektonHub{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: TektonHubSpec{
			Db: DbSpec{
				DbSecretName: "db",
			},
			Api: ApiSpec{
				HubConfigUrl: "https://hubconfigurl.com",
			},
		},
	}

	err := th.Validate(context.TODO())
	assert.Equal(t, "missing field(s): spec.api.ApiSecretName", err.Error())
}

func Test_ValidateTektonHub_InvalidHubConfigUrl(t *testing.T) {

	th := &TektonHub{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: TektonHubSpec{
			Db: DbSpec{
				DbSecretName: "db",
			},
			Api: ApiSpec{
				ApiSecretName: "api",
				HubConfigUrl:  "hubconfigurl",
			},
		},
	}

	err := th.Validate(context.TODO())
	assert.Equal(t, "invalid value: hubconfigurl: spec.api.HubConfigUrl", err.Error())
}
