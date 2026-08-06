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

package pod_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	podstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/pod"
)

var failedPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Failed",
		"conditions": []map[string]any{
			{
				"type":   "Initialized",
				"status": "True",
			},
		},
	},
}

var succeededPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Succeeded",
		"conditions": []map[string]any{
			{
				"type":   "Initialized",
				"status": "True",
			},
		},
	},
}

var runningPodManifest1 = map[string]any{
	"metadata": map[string]any{
		"creationTimestamp": "2026-01-01T10:00:00Z",
	},
	"spec": map[string]any{
		"containers": []any{
			map[string]any{
				"image": "busybox",
			},
		},
	},
	"status": map[string]any{
		"phase": "Running",
		"podIP": "127.0.0.2",
		"podIPs": []any{
			map[string]any{
				"ip": "127.0.0.2",
			},
			map[string]any{
				"ip": "::7f00:0001",
			},
		},
		"conditions": []map[string]any{
			{
				"type":   "Initialized",
				"status": "True",
			},
			{
				"type":   "Ready",
				"status": "True",
			},
		},
		"containerStatuses": []any{
			map[string]any{
				"ready":        true,
				"restartCount": int64(2),
			},
		},
	},
}

var pendingPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Pending",
		"conditions": []map[string]any{
			{
				"type":   "Initialized",
				"status": "False",
			},
		},
	},
}

var terminatingPodManifest = map[string]any{
	"metadata": map[string]any{
		"deletionTimestamp": "2026-01-01T10:00:00Z",
	},
	"status": map[string]any{
		"phase": "Running",
	},
}

var unknownPodManifest1 = map[string]any{
	"metadata": map[string]any{
		"deletionTimestamp": "2026-01-01T10:00:00Z",
	},
	"status": map[string]any{
		"phase":  "Running",
		"reason": "NodeLost",
	},
}

var completedPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Succeeded",
		"containerStatuses": []any{
			map[string]any{
				"state": map[string]any{
					"terminated": map[string]any{
						"reason": "Completed",
					},
				},
			},
		},
	},
}

var waitReasonPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Pending",
		"containerStatuses": []any{
			map[string]any{
				"state": map[string]any{
					"waiting": map[string]any{
						"message": "Error response from daemon: No command specified",
						"reason":  "CreateContainerError",
					},
				},
			},
		},
	},
}

var signalPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Failed",
		"containerStatuses": []any{
			map[string]any{
				"state": map[string]any{
					"terminated": map[string]any{
						"signal": 1,
					},
				},
			},
		},
	},
}

var exitCodePodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Failed",
		"containerStatuses": []any{
			map[string]any{
				"state": map[string]any{
					"terminated": map[string]any{
						"ExitCode": 1,
					},
				},
			},
		},
	},
}

var runningPodManifest2 = map[string]any{
	"status": map[string]any{
		"phase": "Completed",
		"conditions": []map[string]any{
			{
				"type":   "Ready",
				"status": "True",
			},
		},
		"containerStatuses": []any{
			map[string]any{
				"ready": true,
				"state": map[string]any{
					"running": map[string]any{
						"startAt": "2026-01-01T10:00:00Z",
					},
				},
			},
		},
	},
}

var notReadyPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Completed",
		"conditions": []map[string]any{
			{
				"type":   "Ready",
				"status": "False",
			},
		},
		"containerStatuses": []any{
			map[string]any{
				"ready": true,
				"state": map[string]any{
					"running": map[string]any{
						"startAt": "2026-01-01T10:00:00Z",
					},
				},
			},
		},
	},
}

var unknownPodManifest2 = map[string]any{
	"status": map[string]any{
		"phase": nil,
	},
}

var initSignalPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Pending",
		"initContainerStatuses": []any{
			map[string]any{
				"state": map[string]any{
					"terminated": map[string]any{
						"exitCode": 0,
					},
				},
			},
			map[string]any{
				"state": map[string]any{
					"terminated": map[string]any{
						"exitCode": 1,
						"signal":   1,
					},
				},
			},
		},
	},
}

var initExitCodePodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Pending",
		"initContainerStatuses": []any{
			map[string]any{
				"state": map[string]any{
					"terminated": map[string]any{
						"exitCode": 1,
					},
				},
			},
		},
	},
}

var initTermPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Pending",
		"initContainerStatuses": []any{
			map[string]any{
				"state": map[string]any{
					"terminated": map[string]any{
						"exitCode": 1,
						"reason":   "term init",
					},
				},
			},
		},
	},
}

var initWaitPodManifest = map[string]any{
	"status": map[string]any{
		"phase": "Pending",
		"initContainerStatuses": []any{
			map[string]any{
				"state": map[string]any{
					"waiting": map[string]any{
						"reason": "wait init",
					},
				},
			},
		},
	},
}

var initRunPodManifest = map[string]any{
	"spec": map[string]any{
		"initContainers": []any{
			map[string]any{
				"name": "nginx",
			},
		},
	},
	"status": map[string]any{
		"phase": "Pending",
		"initContainerStatuses": []any{
			map[string]any{
				"state": map[string]any{
					"running": map[string]any{
						"startAt": "2026-01-01T10:00:00Z",
					},
				},
			},
		},
	},
}

var _ = Describe("PodStatusParser", func() {
	Context("when parsing basic pod phases", func() {
		It("should return Failed for failed pod", func() {
			parser := podstatus.NewParser(failedPodManifest)
			Expect(parser.Parse().Code).To(Equal("Failed"))
		})

		It("should return Succeeded for succeeded pod", func() {
			parser := podstatus.NewParser(succeededPodManifest)
			Expect(parser.Parse().Code).To(Equal("Succeeded"))
		})

		It("should return Running for running pod with ready containers", func() {
			parser := podstatus.NewParser(runningPodManifest1)
			Expect(parser.Parse().Code).To(Equal("Running"))
		})

		It("should return Pending for pending pod", func() {
			parser := podstatus.NewParser(pendingPodManifest)
			Expect(parser.Parse().Code).To(Equal("Pending"))
		})

		It("should return Terminating for pod with deletion timestamp", func() {
			parser := podstatus.NewParser(terminatingPodManifest)
			Expect(parser.Parse().Code).To(Equal("Terminating"))
		})

		It("should return Unknown for pod with NodeLost reason", func() {
			parser := podstatus.NewParser(unknownPodManifest1)
			Expect(parser.Parse().Code).To(Equal("Unknown"))
		})

		It("should return Unknown for pod with nil phase", func() {
			parser := podstatus.NewParser(unknownPodManifest2)
			Expect(parser.Parse().Code).To(Equal("Unknown"))
		})
	})

	Context("when parsing container statuses", func() {
		It("should return Completed for succeeded pod with completed container", func() {
			parser := podstatus.NewParser(completedPodManifest)
			Expect(parser.Parse().Code).To(Equal("Completed"))
		})

		It("should return waiting reason for container in waiting state", func() {
			parser := podstatus.NewParser(waitReasonPodManifest)
			Expect(parser.Parse().Code).To(Equal("CreateContainerError"))
		})

		It("should return signal info for terminated container with signal", func() {
			parser := podstatus.NewParser(signalPodManifest)
			Expect(parser.Parse().Code).To(Equal("Signal: 1"))
		})

		It("should return exit code for terminated container without reason", func() {
			parser := podstatus.NewParser(exitCodePodManifest)
			Expect(parser.Parse().Code).To(Equal("ExitCode: 1"))
		})

		It("should return Running for pod with running containers", func() {
			parser := podstatus.NewParser(runningPodManifest2)
			Expect(parser.Parse().Code).To(Equal("Running"))
		})

		It("should return NotReady for pod with not ready condition", func() {
			parser := podstatus.NewParser(notReadyPodManifest)
			Expect(parser.Parse().Code).To(Equal("NotReady"))
		})
	})

	Context("when parsing init container statuses", func() {
		It("should return init signal info for terminated init container with signal", func() {
			parser := podstatus.NewParser(initSignalPodManifest)
			Expect(parser.Parse().Code).To(Equal("Init: Signal 1"))
		})

		It("should return init exit code for terminated init container", func() {
			parser := podstatus.NewParser(initExitCodePodManifest)
			Expect(parser.Parse().Code).To(Equal("Init: ExitCode 1"))
		})

		It("should return init reason for terminated init container with reason", func() {
			parser := podstatus.NewParser(initTermPodManifest)
			Expect(parser.Parse().Code).To(Equal("Init: term init"))
		})

		It("should return init reason for waiting init container", func() {
			parser := podstatus.NewParser(initWaitPodManifest)
			Expect(parser.Parse().Code).To(Equal("Init: wait init"))
		})

		It("should return init progress for running init container", func() {
			parser := podstatus.NewParser(initRunPodManifest)
			Expect(parser.Parse().Code).To(Equal("Init: 0/1"))
		})
	})
})
