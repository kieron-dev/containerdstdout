package containerdstdout_test

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// This needs to be run as root so it can contact the containerd service
// Setting path, and maybe gopath is required before running ginkgo:
//
// export GOPATH=/home/vagrant/go
// export PATH=$PATH:/usr/local/go/bin/:$GOPATH/bin
// ginkgo -untilItFails

const containerdSocket = "/run/containerd/containerd.sock"

var _ = Describe("Stdouttest", func() {
	var (
		ctx       context.Context
		client    *containerd.Client
		image     containerd.Image
		container containerd.Container
		task      containerd.Task
		process   containerd.Process
		// processExit <-chan containerd.ExitStatus
	)

	BeforeEach(func() {
		var err error

		client, err = containerd.New(containerdSocket)
		Expect(err).NotTo(HaveOccurred())

		ctx = namespaces.WithNamespace(context.Background(), "example")

		image, err = client.Pull(ctx, "docker.io/library/busybox:latest", containerd.WithPullUnpack)
		Expect(err).NotTo(HaveOccurred())

		container, err = client.NewContainer(
			ctx,
			"example",
			containerd.WithNewSnapshot("busybox-snapshot", image),
			containerd.WithNewSpec(oci.WithImageConfig(image), oci.WithProcessArgs("sleep", "600")),
		)
		Expect(err).NotTo(HaveOccurred())

		task, err = container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
		Expect(err).NotTo(HaveOccurred())

		err = task.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		process.Delete(ctx, containerd.WithProcessKill)
		task.Delete(ctx, containerd.WithProcessKill)
		container.Delete(ctx, containerd.WithSnapshotCleanup)
		client.Close()
	})

	It("captures stdout consistently", func() {
		var err error

		for i := 0; i < 100; i++ {
			stdout := gbytes.NewBuffer()
			process, err = task.Exec(ctx, "say-hello", &specs.Process{
				Args: []string{"/bin/echo", "hi stdout"},
				Cwd:  "/",
			}, cio.NewCreator(
				cio.WithStreams(new(bytes.Buffer), stdout, io.Discard),
				cio.WithFIFODir("/tmp/fifos"),
				cio.WithTerminal,
			))
			Expect(err).NotTo(HaveOccurred())

			// processExit, err = process.Wait(ctx)
			// Expect(err).NotTo(HaveOccurred())

			err = process.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			go exponentialBackoffCloseIO(process, ctx)

			// Eventually(processExit).Should(Receive())
			Eventually(stdout).Should(gbytes.Say("hi stdout"))

			process.Delete(ctx, containerd.WithProcessKill)
		}
	})
})

func exponentialBackoffCloseIO(process containerd.Process, ctx context.Context) {
	duration := 3 * time.Second
	retries := 10
	for i := 0; i < retries; i++ {
		if err := process.CloseIO(ctx, containerd.WithStdinCloser); err != nil {
			time.Sleep(duration)
			duration *= 2
			continue
		}
		break
	}
}
