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

package log_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/log"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

type fakeClient struct {
	initFn     func(context.Context, string, string, string) (*bkciapi.BuildLog, error)
	moreFn     func(context.Context, string, string, string, int64, int64) (*bkciapi.BuildLog, error)
	downloadFn func(context.Context, string, string, string) (io.ReadCloser, error)
}

func (c *fakeClient) GetInitBuildLog(
	ctx context.Context, projectCode, pipelineID, buildID string,
) (*bkciapi.BuildLog, error) {
	if c.initFn == nil {
		return nil, nil
	}
	return c.initFn(ctx, projectCode, pipelineID, buildID)
}

func (c *fakeClient) GetMoreBuildLogs(
	ctx context.Context, projectCode, pipelineID, buildID string, start, batchSize int64,
) (*bkciapi.BuildLog, error) {
	if c.moreFn == nil {
		return nil, nil
	}
	return c.moreFn(ctx, projectCode, pipelineID, buildID, start, batchSize)
}

func (c *fakeClient) DownloadBuildLogs(
	ctx context.Context, projectCode, pipelineID, buildID string,
) (io.ReadCloser, error) {
	if c.downloadFn == nil {
		return nil, nil
	}
	return c.downloadFn(ctx, projectCode, pipelineID, buildID)
}

var _ = Describe("Service", func() {
	var (
		svc       *log.Service
		ctx       context.Context
		stubBkci  *bkciapi.StubApiClient
		fakeBuild *log.BuildLogQuery
	)

	BeforeEach(func() {
		ctx = context.Background()
		stubBkci = bkciapi.NewStub(fakeUser())
		fakeBuild = &log.BuildLogQuery{
			ProjectCode: "stub-project",
			PipelineID:  "stub-pipeline-id-1",
			BuildID:     "b-stub-build-id-test",
			AppID:       "demo-app",
		}
		svc = log.NewService()
	})

	Describe("BuildLogQuery", func() {
		It("builds the default download filename from app and build IDs", func() {
			Expect(fakeBuild.DownloadFilename()).To(Equal("build-log_demo-app_b-stub-build-id-test.log"))
		})
	})

	Describe("StreamLogs", func() {
		It("delivers initial logs and completes when build is finished", func() {
			var chunks []*bkciapi.BuildLog

			err := svc.StreamLogs(ctx, stubBkci, fakeBuild, func(chunk *bkciapi.BuildLog) {
				chunks = append(chunks, chunk)
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(chunks).To(HaveLen(1))
			Expect(chunks[0].Logs).To(HaveLen(5))
			Expect(chunks[0].Logs[0].Message).To(ContainSubstring("Starting build"))
			Expect(chunks[0].Logs[4].Message).To(ContainSubstring("completed successfully"))
		})

		It("completes successfully even if context is cancelled after init (stub returns finished)", func() {
			cancelCtx, cancel := context.WithCancel(ctx)
			var received int

			err := svc.StreamLogs(cancelCtx, stubBkci, fakeBuild, func(chunk *bkciapi.BuildLog) {
				received += len(chunk.Logs)
				cancel()
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(received).To(Equal(5))
		})

		It("returns wrapped error when initial log fetch fails", func() {
			client := &fakeClient{
				initFn: func(context.Context, string, string, string) (*bkciapi.BuildLog, error) {
					return nil, errors.New("init failed")
				},
			}

			err := svc.StreamLogs(ctx, client, fakeBuild, func(*bkciapi.BuildLog) {})

			Expect(err).To(MatchError(ContainSubstring("get init build log")))
			Expect(err).To(MatchError(ContainSubstring("init failed")))
		})

		It("stops immediately when initial response reports EMPTY status", func() {
			var moreCalls int32
			client := &fakeClient{
				initFn: func(context.Context, string, string, string) (*bkciapi.BuildLog, error) {
					return &bkciapi.BuildLog{
						Status:   1,
						Message:  "no logs",
						HasMore:  true,
						Finished: false,
					}, nil
				},
				moreFn: func(context.Context, string, string, string, int64, int64) (*bkciapi.BuildLog, error) {
					atomic.AddInt32(&moreCalls, 1)
					return nil, errors.New("should not fetch more")
				},
			}

			err := svc.StreamLogs(ctx, client, fakeBuild, func(*bkciapi.BuildLog) {})

			Expect(err).NotTo(HaveOccurred())
			Expect(atomic.LoadInt32(&moreCalls)).To(BeZero())
		})

		It("continues streaming from the next cursor after initial logs", func() {
			var starts []int64
			client := &fakeClient{
				initFn: func(context.Context, string, string, string) (*bkciapi.BuildLog, error) {
					return &bkciapi.BuildLog{
						HasMore:  true,
						Finished: false,
						Logs: []bkciapi.LogLine{
							{LineNo: 0, Message: "init-0"},
							{LineNo: 1, Message: "init-1"},
						},
					}, nil
				},
				moreFn: func(_ context.Context, _, _, _ string, start, _ int64) (*bkciapi.BuildLog, error) {
					starts = append(starts, start)
					return &bkciapi.BuildLog{
						HasMore:  false,
						Finished: true,
						Logs: []bkciapi.LogLine{
							{LineNo: 2, Message: "more-2"},
							{LineNo: 3, Message: "more-3"},
						},
					}, nil
				},
			}
			var chunks []*bkciapi.BuildLog

			err := svc.StreamLogs(ctx, client, fakeBuild, func(chunk *bkciapi.BuildLog) {
				chunks = append(chunks, chunk)
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(starts).To(Equal([]int64{2}))
			Expect(chunks).To(HaveLen(2))
			Expect(chunks[0].Logs[0].Message).To(Equal("init-0"))
			Expect(chunks[1].Logs[0].Message).To(Equal("more-2"))
		})

		It("returns wrapped error when fetching incremental logs fails", func() {
			client := &fakeClient{
				initFn: func(context.Context, string, string, string) (*bkciapi.BuildLog, error) {
					return &bkciapi.BuildLog{HasMore: true, Finished: false}, nil
				},
				moreFn: func(context.Context, string, string, string, int64, int64) (*bkciapi.BuildLog, error) {
					return nil, io.ErrUnexpectedEOF
				},
			}

			err := svc.StreamLogs(ctx, client, fakeBuild, func(*bkciapi.BuildLog) {})

			Expect(err).To(MatchError(ContainSubstring("get more build log")))
			Expect(err).To(MatchError(ContainSubstring(io.ErrUnexpectedEOF.Error())))
		})

		It("stops polling when incremental response reports EMPTY status", func() {
			var moreCalls int32
			client := &fakeClient{
				initFn: func(context.Context, string, string, string) (*bkciapi.BuildLog, error) {
					return &bkciapi.BuildLog{
						Status:   0,
						HasMore:  true,
						Finished: false,
						Logs:     []bkciapi.LogLine{{LineNo: 0, Message: "init"}},
					}, nil
				},
				moreFn: func(context.Context, string, string, string, int64, int64) (*bkciapi.BuildLog, error) {
					atomic.AddInt32(&moreCalls, 1)
					return &bkciapi.BuildLog{
						Status:   1,
						Message:  "no logs",
						HasMore:  true,
						Finished: false,
					}, nil
				},
			}

			err := svc.StreamLogs(ctx, client, fakeBuild, func(*bkciapi.BuildLog) {})

			Expect(err).NotTo(HaveOccurred())
			Expect(atomic.LoadInt32(&moreCalls)).To(Equal(int32(1)))
		})

		It("returns context cancellation while waiting at the current log tail", func() {
			cancelCtx, cancel := context.WithCancel(ctx)
			client := &fakeClient{
				initFn: func(context.Context, string, string, string) (*bkciapi.BuildLog, error) {
					return &bkciapi.BuildLog{
						HasMore:  true,
						Finished: false,
						Logs:     []bkciapi.LogLine{{LineNo: 0, Message: "init"}},
					}, nil
				},
				moreFn: func(context.Context, string, string, string, int64, int64) (*bkciapi.BuildLog, error) {
					cancel()
					return &bkciapi.BuildLog{HasMore: false, Finished: false}, nil
				},
			}

			err := svc.StreamLogs(cancelCtx, client, fakeBuild, func(*bkciapi.BuildLog) {})

			Expect(err).To(MatchError(context.Canceled))
		})

		It("backs off when BKCI returns empty logs but build is not complete", func() {
			cancelCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			var moreCalls int32
			client := &fakeClient{
				initFn: func(context.Context, string, string, string) (*bkciapi.BuildLog, error) {
					return &bkciapi.BuildLog{
						HasMore:  true,
						Finished: false,
						Logs:     []bkciapi.LogLine{{LineNo: 0, Message: "init"}},
					}, nil
				},
				moreFn: func(context.Context, string, string, string, int64, int64) (*bkciapi.BuildLog, error) {
					atomic.AddInt32(&moreCalls, 1)
					return &bkciapi.BuildLog{HasMore: true, Finished: false, Logs: nil}, nil
				},
			}

			go func() {
				time.Sleep(20 * time.Millisecond)
				cancel()
			}()

			err := svc.StreamLogs(cancelCtx, client, fakeBuild, func(*bkciapi.BuildLog) {})

			Expect(err).To(MatchError(context.Canceled))
			Expect(atomic.LoadInt32(&moreCalls)).To(Equal(int32(1)))
		})
	})

	Describe("OpenDownloadStream", func() {
		It("uses the dedicated download stream without incremental polling", func() {
			client := &fakeClient{
				moreFn: func(context.Context, string, string, string, int64, int64) (*bkciapi.BuildLog, error) {
					return nil, io.ErrUnexpectedEOF
				},
				downloadFn: func(context.Context, string, string, string) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("[INFO] downloaded from download_logs\n")), nil
				},
			}

			reader, err := svc.OpenDownloadStream(ctx, client, fakeBuild)

			Expect(err).NotTo(HaveOccurred())
			defer reader.Close()

			content, readErr := io.ReadAll(reader)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("[INFO] downloaded from download_logs\n"))
		})

		It("returns readable content from the download stream", func() {
			reader, err := svc.OpenDownloadStream(ctx, stubBkci, fakeBuild)

			Expect(err).NotTo(HaveOccurred())
			defer reader.Close()

			content, readErr := io.ReadAll(reader)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(bytes.Contains(content, []byte("Starting build"))).To(BeTrue())
			Expect(bytes.Contains(content, []byte("completed successfully"))).To(BeTrue())
		})

		It("returns newline-delimited content from the download stream", func() {
			reader, err := svc.OpenDownloadStream(ctx, stubBkci, fakeBuild)

			Expect(err).NotTo(HaveOccurred())
			defer reader.Close()

			content, readErr := io.ReadAll(reader)
			Expect(readErr).NotTo(HaveOccurred())

			lines := bytes.Split(bytes.TrimRight(content, "\n"), []byte("\n"))
			Expect(lines).To(HaveLen(5))
		})

		It("returns a wrapped error when opening the download stream fails", func() {
			client := &fakeClient{
				downloadFn: func(context.Context, string, string, string) (io.ReadCloser, error) {
					return nil, errors.New("download failed")
				},
			}

			reader, err := svc.OpenDownloadStream(ctx, client, fakeBuild)

			Expect(reader).To(BeNil())
			Expect(err).To(MatchError(ContainSubstring("download build log")))
			Expect(err).To(MatchError(ContainSubstring("download failed")))
		})
	})
})
