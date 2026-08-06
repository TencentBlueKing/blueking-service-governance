/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package probe

import (
	"github.com/jinzhu/copier"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Spec stores container probe settings (liveness / readiness / startup).
type Spec struct {
	// Liveness probe configuration.
	Liveness *Probe `bson:"liveness,omitempty"`
	// Readiness probe configuration.
	Readiness *Probe `bson:"readiness,omitempty"`
	// Startup probe configuration.
	Startup *Probe `bson:"startup,omitempty"`
}

// Probe defines the configuration for a single probe.
type Probe struct {
	// Handler defines the action to take.
	Handler *Handler `bson:"probeHandler,omitempty"`
	// InitialDelaySeconds is the number of seconds after the container has started
	// before the probe is initiated.
	InitialDelaySeconds *int32 `bson:"initialDelaySeconds,omitempty"`
	// TimeoutSeconds is the number of seconds after which the probe times out.
	TimeoutSeconds *int32 `bson:"timeoutSeconds,omitempty"`
	// PeriodSeconds is how often (in seconds) to perform the probe.
	PeriodSeconds *int32 `bson:"periodSeconds,omitempty"`
	// SuccessThreshold is the minimum consecutive successes for the probe to be
	// considered successful after having failed.
	SuccessThreshold *int32 `bson:"successThreshold,omitempty"`
	// FailureThreshold is the minimum consecutive failures for the probe to be
	// considered failed after having succeeded.
	FailureThreshold *int32 `bson:"failureThreshold,omitempty"`
}

// Handler defines a specific action that should be taken in a probe.
type Handler struct {
	// Type of action: "EXEC", "HTTP", or "TCP".
	Type string `bson:"_type"`
	// Command is the command line to execute when Type == "EXEC" (mutually exclusive with ShCommand).
	Command []string `bson:"command,omitempty"`
	// ShCommand is the script body for /bin/sh -c when Type == "EXEC" (mutually exclusive with Command).
	ShCommand string `bson:"shCommand,omitempty"`
	// URL is the HTTP endpoint to call (only when Type == "HTTP").
	URL string `bson:"url,omitempty"`
	// Headers are HTTP headers to send (only when Type == "HTTP").
	Headers map[string]string `bson:"headers,omitempty"`
	// Port is the check port (used when Type == "HTTP" or Type == "TCP", range 1~65535).
	Port int32 `bson:"port,omitempty"`
}

// Clone deep-copies the section and collapses empty specs to nil.
func Clone(spec *Spec) *Spec {
	if spec == nil {
		return nil
	}

	cloned := new(Spec)
	_ = copier.CopyWithOption(cloned, spec, copier.Option{DeepCopy: true})
	if !HasData(cloned) {
		return nil
	}
	return cloned
}

// HasData returns whether the section carries any explicit configuration.
func HasData(spec *Spec) bool {
	return spec != nil && (spec.Liveness != nil || spec.Readiness != nil || spec.Startup != nil)
}

// Merge overlays non-nil values from override onto base.
// Each probe type is replaced as a whole (not field-level merge).
func Merge(base, override *Spec) *Spec {
	switch {
	case base == nil && override == nil:
		return nil
	case base == nil:
		return Clone(override)
	case override == nil:
		return Clone(base)
	}

	merged := Clone(base)
	if override.Liveness != nil {
		merged.Liveness = cloneProbe(override.Liveness)
	}
	if override.Readiness != nil {
		merged.Readiness = cloneProbe(override.Readiness)
	}
	if override.Startup != nil {
		merged.Startup = cloneProbe(override.Startup)
	}
	return merged
}

// AppendPatch adds MongoDB $set entries for this section.
func AppendPatch(set *bson.D, spec *Spec) {
	if spec == nil {
		return
	}
	if spec.Liveness != nil {
		*set = append(*set, bson.E{Key: "probes.liveness", Value: spec.Liveness})
	}
	if spec.Readiness != nil {
		*set = append(*set, bson.E{Key: "probes.readiness", Value: spec.Readiness})
	}
	if spec.Startup != nil {
		*set = append(*set, bson.E{Key: "probes.startup", Value: spec.Startup})
	}
}

// cloneProbe deep-copies a single Probe.
func cloneProbe(p *Probe) *Probe {
	if p == nil {
		return nil
	}
	cloned := new(Probe)
	_ = copier.CopyWithOption(cloned, p, copier.Option{DeepCopy: true})
	return cloned
}
