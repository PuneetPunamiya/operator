/*
Copyright 2020 The Tekton Authors

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

package common

import (
	"context"
	"errors"

	mf "github.com/manifestival/manifestival"
	v1alpha1 "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// CheckDeployments checks all deployments in the given manifest and updates the given
// status with the status of the deployments.
func CheckJobs(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.TektonComponent) error {
	th := instance.(*v1alpha1.TektonHub)
	for _, u := range manifest.Filter(mf.ByKind("Job")).Resources() {
		resource, err := manifest.Client.Get(&u)
		if err != nil {
			// change for Job here
			th.Status.MarkDbMigrationInstallerSetNotReady(err.Error())
			return err
		}
		job := &batchv1.Job{}
		if err := scheme.Scheme.Convert(resource, job, nil); err != nil {
			return err
		}
		if !isJobCompleted(job) {
			th.Status.MarkDbMigrationFailed()
			return errors.New("job unsuccessful")
		}
	}

	return nil
}

func isJobCompleted(d *batchv1.Job) bool {
	for _, c := range d.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
