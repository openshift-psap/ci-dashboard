/*
 * Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
 * Copyright (c) 2021, Red Hat.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

const Version = "v1"

type MatricesSpec struct {
	Version string                 `json:"version"`
	Matrices map[string]MatrixSpec `json:"matrices,omitempty"`
}

type TestSpec struct {
	ProwName string        `json:"prow_name,omitempty"`
	OperatorVersion string `json:"operator_version,omitempty"`
	/* *** */
	TestGroup string
	BuildId string
	Passed bool
	Result string
	FinishDate string
}

type MatrixSpec struct {
	Description string        `json:"description,omitempty"`
	ViewerURL string          `json:"viewer_url,omitempty"`
	ArtifactsURL string       `json:"artifacts_url,omitempty"`
	ArtifactsCache string     `json:"artifacts_cache,omitempty"`
	ArtifactsTestName string  `json:"artifacts_test_name,omitempty"`
	OperatorName string       `json:"operator_name,omitempty"`
	Tests map[string][]TestSpec `json:"tests,omitempty"`
}
